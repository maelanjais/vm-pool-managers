<script lang="ts">
import { onDestroy, onMount } from 'svelte';
import { goto } from '$app/navigation';
import { authStore, serverpoolStore } from '$lib/index';
import { P , Button, Dropdown, DropdownItem, Table, TableBody, TableHead, TableBodyCell, TableBodyRow, TableHeadCell } from 'flowbite-svelte';
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
  }
});

onDestroy(() => {
  clearInterval(interval);
});

let selectedsp: string = 'Choisissez le serverpool';

let servers: Server[] = [];
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
</script>

<Button size="md" class="w-48 h-12">{selectedsp}<ChevronDownOutline class="ms-2 h-6 text-white" /></Button>
<Dropdown simple isOpen={false} class="mt-2">
  {#each serverpools as sp}
    <DropdownItem name={sp.serverpool_id} onclick={handleClick}>{sp.serverpool_id}</DropdownItem>
  {/each}
</Dropdown>

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