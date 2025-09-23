package controllers

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

// return a list of all serverpool (might not be useful)
func GetServerpool(c *gin.Context) {

	allServers, err := utils.GetAllServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	var activeServs []gin.H
	for _, s := range allServers {
		activeServs = append(activeServs, gin.H{
			"id":       s.ID,
			"name":     s.Name,
			"HostID":   s.HostID,
			"status":   s.Status,
			"Progress": s.Progress,
		})
	}
	c.JSON(http.StatusOK, gin.H{"servers": activeServs})
}

// create a serverpool in DB, instances will be created by maincrawler
// take only the name of the new serverpool, with authentication before
// create serverpool with base config for now, adding possibles configuration from form
func CreateServerpool(c *gin.Context) {
	//essai avec les meme image et flavor que admin
	userID, exist := c.Get("user_id")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	var body struct {
		Namesp    string   `json:"namesp"`
		ImageRef  string   `json:"image_ref"`
		FlavorRef string   `json:"flavor_ref"`
		Networks  []string `json:"networks"`
		MinVM     int      `json:"min_vm"`
		MaxVM     int      `json:"max_vm"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.Database.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := config.Database.Create(&models.Serverpool{
		UserID:       user.Email,
		ServerpoolID: body.Namesp,
		ImageRef:     body.ImageRef,
		FlavorRef:    body.FlavorRef,
		Networks:     models.JSONStringSlice(body.Networks),
		MinVM:        body.MinVM,
		MaxVM:        body.MaxVM,
		PendingJobs:  0,
	}).Error; err != nil {
		// if err := config.Database.Create(&models.Serverpool{
		// 	UserID:       user.Email,
		// 	ServerpoolID: body.Namesp,
		// 	ImageRef:     os.Getenv("SERVER_IMAGE_REF"),
		// 	FlavorRef:    os.Getenv("SERVER_FLAVOR_REF"),
		// 	Networks:     models.JSONStringSlice{os.Getenv("NETWORK_ID")},
		// 	MinVM:        2,
		// 	MaxVM:        4,
		// 	PendingJobs:  0,
		// }).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create serverpool"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "serverpool created",
		"serverpool": body.Namesp,
	})
}

// delete a serverpool in DB and lauching jobs to delete instances
// takes only serverpool_ID and need to be authenticated
func DeleteServerpool(c *gin.Context) {
	userID, exist := c.Get("user_id")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	var body struct {
		Namesp string `json:"namesp"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.Database.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := config.Database.Where("user_id = ? AND serverpool_id = ?", user.Email, body.Namesp).
		Delete(&models.Serverpool{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot delete serverpool"})
		return
	}

	// pour tout les servers qui ont la paire user.email et body.namesp, creer un job highpriority pour les delete de openstack
	allServers, err := utils.GetAllServers()
	if err != nil {
		return
	}

	for _, ops := range allServers {
		s := models.FromGopherServer(ops)
		if s.UserID == user.Email && s.ServerpoolID == body.Namesp {
			var args []string
			args = append(args, "instance_id")
			args = append(args, s.ID)
			worker.AddJob(*worker.CreateJob(models.DeleteVM, utils.BuildDataMap(args)), true)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "serverpool deleted",
		"serverpool": body.Namesp,
	})
}

func GetMyServerpools(c *gin.Context) {
	userID, exist := c.Get("email")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}
	allsp, err := utils.GetAllServerPool()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve serverpools from Openstack"})
		return
	}

	var ressps []gin.H

	for _, sp := range allsp {
		if sp.UserID == userID {
			ressps = append(ressps, gin.H{
				"serverpool_id": sp.ServerpoolID,
				"image_ref":     sp.ImageRef,
				"flavor_ref":    sp.FlavorRef,
				"min_vm":        sp.MinVM,
				"max_vm":        sp.MaxVM,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"serverpools": ressps})
}

func GetServersInServerpool(c *gin.Context) {
	userEmail, exist := c.Get("email")
	if !exist {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	serverpoolID := c.Param("id")
	if serverpoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing serverpool_id"})
		return
	}

	allServers, err := utils.GetAllServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve servers"})
		return
	}

	var serversInPool []gin.H
	for _, s := range allServers {
		ms := models.FromGopherServer(s)
		if ms.UserID == userEmail && ms.ServerpoolID == serverpoolID {
			serversInPool = append(serversInPool, gin.H{
				"id":        s.ID,
				"name":      s.Name,
				"status":    s.Status,
				"flavor_id": s.Flavor["name"],
				"image_id":  s.Image["name"],
				"addresses": s.Addresses,
				"created":   s.Created,
				"updated":   s.Updated,
				"host_id":   s.HostID,
				"progress":  s.Progress,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"servers": serversInPool})
}

func GetAllImages(ctx *gin.Context) {
	opts := &clientconfig.ClientOpts{
		Cloud: os.Getenv("OPTS_CLOUD"),
	}

	client, err := clientconfig.NewServiceClient(ctx, "image", opts)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	allPages, err := images.List(client, images.ListOpts{}).AllPages(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"images": allImages})
}

func GetallFlavors(ctx *gin.Context) {
	opts := &clientconfig.ClientOpts{
		Cloud: os.Getenv("OPTS_CLOUD"),
	}

	client, err := clientconfig.NewServiceClient(ctx, "compute", opts)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	allPages, err := flavors.ListDetail(client, flavors.ListOpts{}).AllPages(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}
	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"flavors": allFlavors})
}

func GetAllNetworks(ctx *gin.Context) {
	opts := &clientconfig.ClientOpts{
		Cloud: os.Getenv("OPTS_CLOUD"),
	}

	client, err := clientconfig.NewServiceClient(ctx, "network", opts)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	allPages, err := networks.List(client, networks.ListOpts{}).AllPages(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	allNets, err := networks.ExtractNetworks(allPages)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "user not connected"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"networks": allNets})
}
