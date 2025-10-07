<script lang="ts">
  import { authStore, serverpoolStore } from '$lib/index';
  import { goto } from '$app/navigation';
  import { Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Button } from "flowbite-svelte";

  let token: string | null = null;
  $: token = $authStore;

  let user;
  let serverpools;
  let error;
  $: ({ user, serverpools, error } = $serverpoolStore);

</script>

<Table shadow hoverable={true} class="w-full text-tertiary-50">
  {#if error}
    <p class="text-red-500">{error}</p>
  {:else}
    <caption class="text-2xl text-left font-bold mb-4 pl-4">
      Profil de l'utilisateur
      <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>Nom :</strong> {user?.name}</p>
      <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>Email :</strong> {user?.email}</p>
    </caption>

    {#if !serverpools || serverpools.length === 0}
      <p class="text-gray-500">Aucun serverpool trouvé</p>
    {:else}
      <TableHead class="bg-secondary-200">
        <TableHeadCell>Serverpool Name</TableHeadCell>
        <TableHeadCell>Image</TableHeadCell>
        <TableHeadCell>Flavor</TableHeadCell>
        <TableHeadCell>Minimum VM</TableHeadCell>
        <TableHeadCell>Maximum VM</TableHeadCell>
        <TableHeadCell><span class="sr-only">Inspect</span></TableHeadCell>
      </TableHead>
      <TableBody>
        {#each serverpools as sp, i}
          <TableBodyRow class={i % 2 === 0 ? 'bg-tertiary-400 hover:bg-tertiary-200' : 'bg-tertiary-300 hover:bg-tertiary-200'}>
            <TableBodyCell>{sp.serverpool_id}</TableBodyCell>
            <TableBodyCell>{sp.image_ref}</TableBodyCell>
            <TableBodyCell>{sp.flavor_ref}</TableBodyCell>
            <TableBodyCell>{sp.min_vm}</TableBodyCell>
            <TableBodyCell>{sp.max_vm}</TableBodyCell>
            <TableBodyCell class="flex justify-center"><Button class="bg-option-500"onclick={() => goto(`/serverpools/${sp.serverpool_id}`)}>Inspect</Button></TableBodyCell>
          </TableBodyRow>
        {/each}
      </TableBody>
    {/if}
  {/if}
</Table>
