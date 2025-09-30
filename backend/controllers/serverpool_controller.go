package controllers

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
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

	serverpoolID := c.Param("id")
	if serverpoolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing serverpool_id"})
		return
	}

	var user models.User
	if err := config.Database.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := config.Database.Where("user_id = ? AND serverpool_id = ?", user.Email, serverpoolID).
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
		if s.UserID == user.Email && s.ServerpoolID == serverpoolID {
			var args []string
			args = append(args, "instance_id")
			args = append(args, s.ID)
			worker.AddJob(*worker.CreateJob(models.DeleteVM, utils.BuildDataMap(args)), true)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "serverpool deleted",
		"serverpool": serverpoolID,
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
				"id":     s.ID,
				"name":   s.Name,
				"status": s.Status,
				"flavor": gin.H{
					"id":   s.Flavor["id"],
					"name": s.Flavor["name"],
				},
				"image": gin.H{
					"id":   s.Image["id"],
					"name": s.Image["name"],
				},
				"addresses": s.Addresses,
				"created":   s.Created,
				"updated":   s.Updated,
				"host_id":   s.HostID,
				"progress":  s.Progress,
			})

		}
	}
	// fmt.Println("SERVERS IN POOL:", serversInPool)
	c.JSON(http.StatusOK, gin.H{"servers": serversInPool})
}

func GetAllImages(c *gin.Context) {
	var images []models.Image
	if err := config.Database.Find(&images).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch images"})
		return
	}
	c.JSON(http.StatusOK, images)
}

func GetallFlavors(c *gin.Context) {
	var flavor []models.Flavor
	if err := config.Database.Find(&flavor).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch flavors"})
		return
	}
	c.JSON(http.StatusOK, flavor)
}

func GetAllNetworks(c *gin.Context) {
	var networks []models.Network
	if err := config.Database.Find(&networks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch networks"})
		return
	}
	c.JSON(http.StatusOK, networks)
}

type GroupRequest struct {
	Group string `json:"group"`
}

func GetGroupeImage(c *gin.Context) {
	var req GroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var images []models.Image
	if err := config.Database.Find(&images).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch images"})
		return
	}

	var filtered []models.Image
	for _, img := range images {
		named := strings.ToLower(utils.FirstLetters(img.Name))
		if named == strings.ToLower(req.Group) {
			filtered = append(filtered, img)
		}
	}

	c.JSON(http.StatusOK, filtered)
}

type GroupeImageName struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func GetGroupeImagename(c *gin.Context) {
	groupMap := make(map[string][]string)
	var images []models.Image

	if err := config.Database.Find(&images).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch images"})
		return
	}

	for _, img := range images {
		named := strings.ToLower(utils.FirstLetters(img.Name))
		groupMap[named] = append(groupMap[named], img.Name)
	}

	var groupList []GroupeImageName
	for k := range groupMap {
		groupList = append(groupList, GroupeImageName{Name: k, Value: k})
	}

	c.JSON(http.StatusOK, groupList)
}

type RebuildRequest struct {
	ServerID   string `json:"serverId" binding:"required"`
	ServerName string `json:"server_name" binding:"required"`
	ImageID    string `json:"image_id" binding:"required"`
}

func RebuildServer(c *gin.Context) {
	var req RebuildRequest

	// Lire le JSON du body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Récupérer user_id / email injectés par le middleware
	userID, _ := c.Get("user_id")
	email, _ := c.Get("email")

	// Créer un client Compute via clouds.yaml
	opts := &clientconfig.ClientOpts{
		Cloud: os.Getenv("OPTS_CLOUD"), // ex: "devstack", "ovh", etc.
	}

	client, err := clientconfig.NewServiceClient(c, "compute", opts)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "failed to create compute client",
			"details": err.Error(),
		})
		return
	}

	// Préparer les options de rebuild
	rebuildOpts := servers.RebuildOpts{
		ImageRef: req.ImageID,
		Name:     req.ServerName,
	}

	// Exécuter le rebuild
	_, err = servers.Rebuild(c.Request.Context(), client, req.ServerID, rebuildOpts).Extract()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to rebuild server",
			"details": err.Error(),
		})
		return
	}

	// Réponse au frontend
	c.JSON(http.StatusOK, gin.H{
		"message":     "rebuild launched successfully",
		"server_id":   req.ServerID,
		"server_name": req.ServerName,
		"image_id":    req.ImageID,
		"user_id":     userID,
		"email":       email,
	})
}
