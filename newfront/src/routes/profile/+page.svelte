<script lang="ts">
  import { authStore, serverPools } from '$lib/store';
  
  import { goto } from '$app/navigation';
  import { Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Button } from "flowbite-svelte";

  let token: string | null = null;
  $: token = $authStore?.token ?? null;

</script>

<Table shadow hoverable={true} class="w-full text-tertiary-50">
    <caption class="text-2xl text-left font-bold mb-4 pl-4">
      Profil de l'utilisateur
      <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>Email :</strong> {$authStore?.email}</p>
    </caption>

    {#if !$serverPools || $serverPools.length === 0}
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
        {#each $serverPools as sp, i}
          <TableBodyRow class={i % 2 === 0 ? 'bg-tertiary-400 hover:bg-tertiary-200' : 'bg-tertiary-300 hover:bg-tertiary-200'}>
            <TableBodyCell>{sp.name}</TableBodyCell>
            <TableBodyCell>{sp.image}</TableBodyCell>
            <TableBodyCell>{sp.flavor}</TableBodyCell>
            <TableBodyCell>{sp.minVm}</TableBodyCell>
            <TableBodyCell>{sp.maxVm}</TableBodyCell>
            <!-- <TableBodyCell class="flex justify-center"><Button class="bg-option-500"onclick={() => goto(`/serverpools/${sp.serverpool_id}`)}>Inspect</Button></TableBodyCell> -->
          </TableBodyRow>
        {/each}
      </TableBody>
    {/if}
</Table>
