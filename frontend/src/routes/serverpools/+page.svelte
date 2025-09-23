<script lang="ts">
import { onDestroy, onMount } from 'svelte';
import { goto } from '$app/navigation';
import { authStore, serverpoolStore, createServerpool , fetchAllImages, fetchAllFlavors , fetchAllNetworks} from '$lib/index';
import type { ImageOption , FlavorOption , NetworkOption } from '$lib/index';
import { Button, Dropdown, DropdownItem, Table, TableBody, TableHead, TableBodyCell, TableBodyRow, TableHeadCell, Modal , Label, Input, Select , Range, Checkbox } from 'flowbite-svelte';
import { ChevronDownOutline } from 'flowbite-svelte-icons';


// Typage pour un serveur
interface Server {
  id: string;
  name: string;
  status: string;
  flavor_id: string;
  image_id: string;
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

onMount(async () => {
  if (!token) {
    goto('/'); // redirige si pas connecté
    return;
  } else {
    serverpoolStore.fetchServerpools();
    interval = setInterval(serverpoolStore.fetchServerpools, 50000);
    const apiImages = await fetchAllImages();
    images = apiImages
      .filter(img => img.status === 'active')
      .map(img => ({
        value: img.value,
        name: img.name || img.value,
        status: img.status
      }));
    console.log("Images disponibles :", images);

    const apiFlavors = await fetchAllFlavors();
    flavors = apiFlavors.map(flavor => ({
      value: flavor.value,
      name: flavor.name || flavor.value
    }));
    console.log("Flavors disponibles :", flavors);

    const apiNetworks = await fetchAllNetworks();
    networks = apiNetworks.map(net => ({
      value: net.value,
      name: net.name || net.value
    }));
    console.log("Réseaux disponibles :", networks);
  }
});

onDestroy(() => {
  clearInterval(interval);
});

let servers: Server[] = [];
let selectedsp: string = 'Choisissez le serverpool';
let selectedImage: string = '';
let selectedFlavor: string = '';
let selectedNetwork: string = '';

let loadingServers = false;

// Quand on clique sur un item du dropdown, on charge les serveurs du pool sélectionné
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
    // 2. Nettoyer la liste des réseaux
    const networks = networksStr.split(',').map(n => n.trim()).filter(n => n);

    // 3. Créer le serverpool avec le flavor.id
    await createServerpool({
      namesp,
      image_ref,
      flavor_ref,
      networks,
      min_vm,
      max_vm
    });

    // 4. Succès
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

</script>



<!-- Dropdown pour choisir un serverpool -->
<Button size="md" class="w-48 h-12">{selectedsp}<ChevronDownOutline class="ms-2 h-6 text-white" /></Button>
<Dropdown simple isOpen={false} class="mt-2">
  {#each serverpools as sp}
    <DropdownItem name={sp.serverpool_id} onclick={handleClick}>{sp.serverpool_id}</DropdownItem>
  {/each}
</Dropdown>

<!-- Table des serveurs -->
{#if loadingServers}
  <p>Chargement des serveurs...</p>
{:else if servers.length}
  <Table>
    <TableHead>
      <TableHeadCell>Nom</TableHeadCell>
      <TableHeadCell>Status</TableHeadCell>
      <TableHeadCell>Flavor</TableHeadCell>
      <TableHeadCell>Image</TableHeadCell>
      <TableHeadCell>IP</TableHeadCell>
      <TableHeadCell>Créé le</TableHeadCell>
    </TableHead>
    <TableBody>
      {#each servers as s}
        <TableBodyRow>
          <TableBodyCell>{s.name}</TableBodyCell>
          <TableBodyCell>{s.status}</TableBodyCell>
          <TableBodyCell>{s.flavor_id}</TableBodyCell>
          <TableBodyCell>{s.image_id}</TableBodyCell>
          <TableBodyCell>
            {#if s.addresses}
              {#each Object.values(s.addresses) as net}
                {#each net as addr}
                  {addr.addr}{' '}
                {/each}
              {/each}
            {/if}
          </TableBodyCell>
          <TableBodyCell>{s.created}</TableBodyCell>
        </TableBodyRow>
      {/each}
    </TableBody>
  </Table>
{:else}
  <p>Aucun serveur trouvé pour ce serverpool.</p>
{/if}



<!-- Formulaire pour creer un serverpool  -->
<Button size="md" color="green" class="mt-4" onclick={() => createspModal = true}>Créer un serverpool</Button>

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
      <Select name="image_ref" items={images} bind:value={selectedImage} required />
    </Label>
    <Label class="space-y-2 text-xl">
      <span>Flavor Ref</span>
      <Select name="flavor_ref" items={flavors} bind:value={selectedFlavor} required />
    </Label>
    <Label class="space-y-2 text-xl">
      <span>Réseaux</span>
      <Select name="networks" items={networks} bind:value={selectedNetwork} required />
    </Label>
    <Label class="space-y-2 text-xl">
      <span>Min VM</span>
      <Input type="number" name="min_vm" min="1" value="1" required />
    </Label>
    <Label class="space-y-2 text-xl">
      <span>Max VM</span>
      <Input type="number" name="max_vm" min="1" value="1" required />
    </Label>
    <Button type="submit" color="green">Créer</Button>
  </form>
</Modal>
{/if}