<script lang="ts">
  import { authStore, serverpoolStore } from '$lib/index';
  import { onDestroy, onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Button } from "flowbite-svelte";

  let token: string | null = null;
  $: token = $authStore;

  let user;
  let serverpools;
  let error;
  $: ({ user, serverpools, error } = $serverpoolStore);

  let interval: ReturnType<typeof setInterval>;

  onMount(() => {
    if (token) {
      serverpoolStore.fetchServerpools();
      interval = setInterval(serverpoolStore.fetchServerpools, 50000);
    } else {
      goto('/'); // redirige si pas connecté
      clearInterval(interval);
    }
  });
  onDestroy(() => {
    clearInterval(interval);
  });
</script>

<Table striped={true} color={'gray'} shadow hoverable={true}>
  {#if error}
    <p class="text-red-500">{error}</p>
  {:else}
    <caption class="text-2xl text-left font-bold mb-4 pl-4">
      Profil de l'utilisateur
      <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>ID :</strong> {user?.id}</p>
      <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>Nom :</strong> {user?.name}</p>
      <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>Email :</strong> {user?.email}</p>
    </caption>

    {#if !serverpools || serverpools.length === 0}
      <p class="text-gray-500">Aucun serverpool trouvé</p>
    {:else}
      <TableHead>
        <TableHeadCell>Serverpool Name</TableHeadCell>
        <TableHeadCell>Image</TableHeadCell>
        <TableHeadCell>Flavor</TableHeadCell>
        <TableHeadCell>Minimum VM</TableHeadCell>
        <TableHeadCell>Maximum VM</TableHeadCell>
        <TableHeadCell><span class="sr-only">Inspect</span></TableHeadCell>
      </TableHead>
      <TableBody>
        {#each serverpools as sp}
          <TableBodyRow>
            <TableBodyCell>{sp.serverpool_id}</TableBodyCell>
            <TableBodyCell>{sp.image_ref}</TableBodyCell>
            <TableBodyCell>{sp.flavor_ref}</TableBodyCell>
            <TableBodyCell>{sp.min_vm}</TableBodyCell>
            <TableBodyCell>{sp.max_vm}</TableBodyCell>
            <TableBodyCell class="flex justify-center"><Button onclick={() => goto(`/serverpools/${sp.serverpool_id}`)}>Inspect</Button></TableBodyCell>
          </TableBodyRow>
        {/each}
      </TableBody>
    {/if}
  {/if}
</Table>
