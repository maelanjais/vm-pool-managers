<script lang="ts">
import { onDestroy, onMount } from 'svelte';
import { goto } from '$app/navigation';
import { authStore, serverpoolStore, createServerpool , fetchAllImages, fetchAllFlavors , fetchAllNetworks, deleteServerpool, rebuildServer, fetchGroupImages , fetchGroupImageName } from '$lib/index';
import type { ImageOption , FlavorOption , NetworkOption , ImageGroupe, GroupedImageOption , GroupeImageName} from '$lib/index';
import { Button, Dropdown, DropdownItem, Table, TableBody, TableHead, TableBodyCell, TableBodyRow, TableHeadCell, Modal , Label, Input, Select , MultiSelect } from 'flowbite-svelte';
import { ChevronDownOutline } from 'flowbite-svelte-icons';
import { page } from '$app/stores';

// Typage serveur
interface Server {
  id: string;
  name: string;
  status: string;
  flavor: { id: string; name: string | null };
  image: { id: string; name: string | null };
  addresses: Record<string, { addr: string }[]>;
  created: string;
  updated?: string;
  host_id?: string;
  progress?: number;
}

let images: ImageOption[] = [];
let flavors: FlavorOption[] = [];
let networks: NetworkOption[] = [];
let token: string | null = null;
$: token = $authStore;

let serverpools;
$: ({ user, serverpools, error } = $serverpoolStore);

let interval: ReturnType<typeof setInterval>;
let selectedsp: string = 'Choisissez le serverpool';
let groupimagename: GroupeImageName[] = [];

onMount(async () => {
  if (!token) {
    goto('/'); 
    return;
  } else {
    serverpoolStore.fetchServerpools();
    interval = setInterval(serverpoolStore.fetchServerpools, 50000);
    loadingServers = true;
    
    const apiImages = await fetchAllImages();
    images = apiImages
    .filter(img => img.status === 'active')
    .map(img => ({
      value: img.value,
      name: img.name || img.value,
      status: img.status,
      Mindisk: img.Mindisk,
      Minram: img.Minram
    }));
    
    const apiFlavors = await fetchAllFlavors();
    flavors = apiFlavors.map(flavor => ({
      value: flavor.value,
      name: flavor.name || flavor.value,
      disk: flavor.disk,
      ram: flavor.ram,
      vcpus: flavor.vcpus,
      rxtx_factor: flavor.rxtx_factor
    }));
    
    
    const apiNetworks = await fetchAllNetworks();
    networks = apiNetworks.map(net => ({
      value: net.value,
      name: net.name || net.value
    }));

    const apiGroupImageName = await fetchGroupImageName();
    groupimagename = apiGroupImageName;

    selectedsp = $page.params.id || 'Choisissez le serverpool';
    await handleSelectServerpool(selectedsp);
  }


});

onDestroy(() => {
  clearInterval(interval);
});

let servers: Server[] = [];

let loadingServers = false;

const handleClick = async (e: Event) => {
  e.preventDefault();
  const target = e.target as HTMLButtonElement;
  selectedsp = target.name;
  await handleSelectServerpool(selectedsp);
};

async function handleSelectServerpool(serverpoolId: string) {
  loadingServers = true;
  servers = await serverpoolStore.fetchServersInServerpool(serverpoolId);
  loadingServers = false;
}

let createspModal = false;
let createError = "";
let createSuccess = false;

async function handleCreateServerpool(event: Event) {
  event.preventDefault();
  const form = event.target as HTMLFormElement;
  const data = new FormData(form);

  createError = "";
  createSuccess = false;

  const namesp = data.get('namesp') as string;
  const image_ref = data.get('image_ref') as string;
  const flavor_ref = data.get('flavor_ref') as string;
  const networksStr = data.get('networks') as string;
  const min_vm = Number(data.get('min_vm'));
  const max_vm = Number(data.get('max_vm'));

  if (!namesp || !image_ref || !flavor_ref || !networksStr || !min_vm || !max_vm) {
    createError = "Tous les champs sont requis";
    return;
  }

  try {
    const networks = networksStr.split(',').map(n => n.trim()).filter(n => n);

    await createServerpool({
      namesp,
      image_ref,
      flavor_ref,
      networks: selectedNetworks,
      min_vm,
      max_vm
    });

    createSuccess = true;
    setTimeout(() => {
      form.reset();
      createspModal = false;
      createSuccess = false;
    }, 3000);

  } catch (err: any) {
    createError = err.message || "Erreur lors de la création du serverpool";
  }
}

async function handleDeleteServerpool(serverpoolId: string) {
  if (!confirm(`Êtes-vous sûr de vouloir supprimer le serverpool ${serverpoolId} ?`)) {
    return;
  }
  try {
    await deleteServerpool(serverpoolId);
    if (selectedsp === serverpoolId) {
      selectedsp = 'Choisissez le serverpool';
      servers = [];
    }
  } catch (err: any) {
    alert(err.message || "Erreur lors de la suppression du serverpool");
  }
}

async function handleRebuildServer(server: Server) {
  if (!confirm(`Êtes-vous sûr de vouloir rebuild le serveur ${server.name} (${server.id}) ?`)) {
    return;
  }
  try {
    await rebuildServer(server.id, server.name, server.image.id);
    alert(`Rebuild du serveur ${server.name} (${server.id}) lancé avec succès.`);
    await handleSelectServerpool(selectedsp);
  } catch (err: any) {
    alert(err.message || "Erreur lors du rebuild du serveur");
  }
}

// Helpers
function getFlavorNameById(id: string): string {
  const flavor = flavors.find(f => f.value === id);
  return flavor ? flavor.name : id;
}

function getImageNameById(id: string): string {
  const img = images.find(i => i.value === id);
  return img ? img.name : id;
}

let selectedNetworks: string[] = [];
let selectedFlavor: string = "";
let selectedImage: string = "";
let selectedGroupImage: string = "";

$: if (selectedGroupImage) {
  fetchGroupImages(selectedGroupImage).then(data => {
    images = data;
    selectedImage = '';
  });
}

$: if (!createspModal){
  selectedFlavor = "";
  selectedGroupImage = "";
  selectedImage = "";
  selectedNetworks = [];
}

</script>

<!-- Dropdown -->
<Button size="md" class="w-48 h-12">
  {selectedsp}<ChevronDownOutline class="ms-2 h-6 text-white" />
</Button>
<Dropdown simple isOpen={false} class="mt-2">
  {#each serverpools as sp}
    <DropdownItem name={sp.serverpool_id} onclick={handleClick}>{sp.serverpool_id}</DropdownItem>
  {/each}
</Dropdown>

<!-- Table -->
{#if loadingServers}
  <p>Chargement des serveurs...</p>
{:else if servers.length > 0}
  <Table hoverable={true} striped={false} class="mt-4 w-full text-tertiary-50">
  <caption class="text-left mb-2">
    {selectedsp}
    <p class="text-sm font-normal">Flavor: {getFlavorNameById(servers[0].flavor.id)}</p>
    <p class="text-sm font-normal">Image: {getImageNameById(servers[0].image.id)}</p>
    <!-- <p class="text-sm font-normal">Networks: {getNetworkNamesByIds(servers[0].networks)}</p> -->
  </caption>

  <TableHead class="bg-tertiary-500 text-white">
    <TableHeadCell>Nom</TableHeadCell>
    <TableHeadCell>Status</TableHeadCell>
    <TableHeadCell>IP</TableHeadCell>
    <TableHeadCell>Créé le</TableHeadCell>
    <TableHeadCell></TableHeadCell>
  </TableHead>

  <TableBody>
    {#each servers as s, i}
      <TableBodyRow class={i % 2 === 0 ? 'bg-tertiary-400 hover:bg-tertiary-200' : 'bg-tertiary-300 hover:bg-tertiary-200'}>
        <TableBodyCell>{s.name}</TableBodyCell>
        <TableBodyCell>{s.status}</TableBodyCell>
        <TableBodyCell>
          {#if s.addresses}
            {#each Object.values(s.addresses) as net}
              {#each net as addr}
                {addr.addr}{';  '}
              {/each}
            {/each}
          {/if}
        </TableBodyCell>
        <TableBodyCell>{s.created}</TableBodyCell>
        <TableBodyCell>
          <Button size="sm" class="bg-option-500" onclick={() => handleRebuildServer(s)}>Rebuild</Button>
        </TableBodyCell>
      </TableBodyRow>
    {/each}
  </TableBody>
</Table>

<Button class="bg-tertiary-500 mt-4" onclick={() => handleDeleteServerpool(selectedsp)}>
  Supprimer le serverpool
</Button>

{:else}
  <p>Aucun serveur trouvé pour ce serverpool.</p>
{/if}

<!-- Modal -->
<Button size="md" class="bg-option-500 mt-4" onclick={() => createspModal = true}>Créer un serverpool</Button>

{#if createspModal}
  <Modal bind:open={createspModal} class="bg-gray-400" focustrap={true}>
    <form class="flex flex-col space-y-6" on:submit|preventDefault={handleCreateServerpool}>
      <h3 class="mb-4 text-2xl font-medium text-gray-800">Créer un Serverpool</h3>
      {#if createError}
        <Label color="red">{createError}</Label>
      {/if}
      {#if createSuccess}
        <Label color="green" class="text-xl">Serverpool créé avec succès</Label>
      {/if}
      <Label class="space-y-2 text-xl">
        <span>Nom du Serverpool</span>
        <Input type="text" name="namesp" placeholder="Nom du serverpool" required />
      </Label>
      <Label class="space-y-2 text-xl">
        <span>Image Ref</span>
        <Select name="image_group" items={groupimagename} required bind:value={selectedGroupImage} />
        {#if selectedGroupImage}
          <Select name="image_ref" items={images} required bind:value={selectedImage} />
          {#each images.filter(img => img.value === selectedImage) as img}
            <p>Status: {img.status}</p>
            <p>Min Disk: {img.Mindisk} GB</p>
            <p>Min RAM: {img.Minram} MB</p>
          {/each}
        {/if}
      </Label>
      <Label class="space-y-2 text-xl">
        <span>Flavor Ref</span>
        <Select name="flavor_ref" items={flavors} bind:value={selectedFlavor} required />
        {#if selectedFlavor}
          {#each flavors.filter(f => f.value === selectedFlavor) as flavor}
            <p>Disk: {flavor.disk} GB</p>
            <p>RAM: {flavor.ram} MB</p>
            <p>vCPUs: {flavor.vcpus}</p>
            <p>RXTX Factor: {flavor.rxtx_factor}</p>
          {/each}
        {/if}
      </Label>
        <span>Réseaux</span>
        <MultiSelect name="networks" bind:value={selectedNetworks} items={networks} placeholder="Sélectionnez les réseaux" required class="bg-gray-200" />
        {#if selectedNetworks.length === 0}
          <p class="text-sm text-gray-500">Aucun réseau sélectionné</p>
        {/if}
        <p>{selectedNetworks}</p>
      <Label class="space-y-2 text-xl">
        <span>Min VM</span>
        <Input type="number" name="min_vm" min="1" value="1" required />
      </Label>
      <Label class="space-y-2 text-xl">
        <span>Max VM</span>
        <Input type="number" name="max_vm" min="1" value="1" required />
      </Label>
      <Button type="submit" class="bg-option-500">Créer</Button>
    </form>
  </Modal>
{/if}
