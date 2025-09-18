<script lang="ts">
  import { authStore } from '$lib/index';
  import { onDestroy, onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Checkbox, TableSearch, tableHeadCell, Button } from "flowbite-svelte";

let token: string | null = null;
  let user: { id?: number; name?: string; email?: string } = {};
  let serverpools: {
    serverpool_id: string;
    image_ref: string;
    flavor_ref: string;
    min_vm: number;
    max_vm: number;
  }[] = [];
  let error: string = "";

  $: token = $authStore;

  async function fetchServerpool() {
    try {
      const res = await fetch('http://localhost:8080/users/me', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!res.ok) {
        error = "Impossible de récupérer le profil";
        return;
      }

      const data = await res.json();
      user = { id: data.id, name: data.name, email: data.email };
   
      const res2 = await fetch('http://localhost:8080/serverpool/mysp', {
        headers : {'Authorization' : `Bearer ${token}`}
      });

      if (!res2.ok) {
        error = "Impossible de récupérer les serverpools";
        return;
      }

      const data2 = await res2.json();
      serverpools = data2.serverpools || [];

    } catch (err) {
      error = "Erreur backend";
      console.error(err);
    }
    
  }

  let interval: ReturnType<typeof setInterval>;

  onMount(async () => {
    if (!token) {
      goto('/'); // redirige si pas connecté
      return;
    } else {
      fetchServerpool();
      interval = setInterval(fetchServerpool, 50000);
    }
  });

  onDestroy(() => {
    clearInterval(interval);
  });
</script>

<Table striped={true} color="gray" shadow hoverable={true}>
  {#if error}
    <p class="text-red-500">{error}</p>
  {:else}
  <caption class="text-2xl text-left font-bold mb-4 pl-4 bg-gray-700">Profil de l'utilisateur
    <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>ID :</strong> {user.id}</p>
    <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>Nom :</strong> {user.name}</p>
    <p class="mt-1 text-sm font-normal text-gray-300 dark:text-gray-400"><strong>Email :</strong> {user.email}</p>
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
            <TableBodyCell class="flex justify-center"><Button>Inspect</Button></TableBodyCell>
          </TableBodyRow>
          {/each}
        </TableBody>
    {/if}
  {/if}
</Table>