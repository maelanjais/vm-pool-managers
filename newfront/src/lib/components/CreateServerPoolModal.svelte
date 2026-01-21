<script lang="ts">
  import {
    Modal,
    Button,
    Label,
    Input,
    Select
  } from 'flowbite-svelte';

  import type { Image, Flavor, Network, Config } from '$lib/type';

  export let open: boolean;
  
  export let images: Image[];
  export let flavors: Flavor[];
  export let networks: Network[];
  export let configs: Config[];

  export let selectedGroupImage: string | null;
  export let selectedImage: string | null;
  export let selectedFlavor: string;
  export let selectedNetwork: string;
  export let selectedConfigFile: string;
  export let scheduleDay: string;
  export let scheduleTime: string;
  export let scheduleWindowHours: number;

  export let createError: string;
  export let createSuccess: boolean;

  export let handleCreateServerpool: (e: Event) => void;
  export let getUniqueFirstAlphaBlocks: (images: Image[]) => string[];
  export let filterImagesByPrefix: (images: Image[], prefix: string) => Image[];
</script>

<Modal bind:open class="bg-gray-500 bg-opacity-50" focustrap>
  <form
    class="flex flex-col space-y-6 p-6 bg-white rounded-lg"
    on:submit|preventDefault={handleCreateServerpool}
  >
    <h3 class="text-xl font-medium text-gray-800">
      Créer un Serverpool
    </h3>

    {#if createError}
      <p class="text-red-500">{createError}</p>
    {/if}

    {#if createSuccess}
      <p class="text-green-600 font-semibold">Serverpool créé !</p>
    {/if}

    <Label>
      <span>Nom du Serverpool</span>
      <Input type="text" name="namesp" required />
    </Label>

    <Label>
      <span>Image</span>

      <Select bind:value={selectedGroupImage} required>
        <option disabled selected value="">
          Choisir un groupe d’images
        </option>
        {#each getUniqueFirstAlphaBlocks(images) as prefix}
          <option value={prefix}>{prefix}</option>
        {/each}
      </Select>

      {#if selectedGroupImage}
        <Select bind:value={selectedImage} required>
          <option disabled selected value="">
            Choisir une image
          </option>
          {#each filterImagesByPrefix(images, selectedGroupImage) as img}
            <option value={img.id}>{img.name}</option>
          {/each}
        </Select>
      {/if}
    </Label>

    <Label>
      <span>Flavor</span>
      <Select bind:value={selectedFlavor} required>
        <option disabled selected value="">Choisir un flavor</option>
        {#each flavors as f}
          <option value={f.id}>{f.name}</option>
        {/each}
      </Select>
    </Label>

    <Label>
      <span>Réseaux</span>
      <Select bind:value={selectedNetwork} required>
        <option disabled selected value="">Choisir un réseau</option>
        {#each networks as n}
          <option value={n.id}>{n.name}</option>
        {/each}
      </Select>
    </Label>

    <Label>
      <span>Min VM</span>
      <Input type="number" name="min_vm" min="1" value="1" required />
    </Label>
    <Label>
      <span>Max VM</span>
      <Input type="number" name="max_vm" min="1" value="1" required />
    </Label>

    <Label>
      <span>Config</span>
      <Select bind:value={selectedConfigFile}>
        <option value="">Défaut</option>
        {#each configs as c}
          <option value={c.name}>{c.name}</option>
        {/each}
      </Select>
    </Label>

    

    <Label>
      <span>Schedule</span>
      <div class="grid grid-cols-3 gap-3 mt-1">
        <Select bind:value={scheduleDay} required>
          <option disabled selected value="">Jour</option>
          <option value="1">Lundi</option>
          <option value="2">Mardi</option>
          <option value="3">Mercredi</option>
          <option value="4">Jeudi</option>
          <option value="5">Vendredi</option>
          <option value="6">Samedi</option>
          <option value="0">Dimanche</option>
        </Select>

        <Input type="time" bind:value={scheduleTime} required />

        <Input
          type="number"
          min="1"
          max="24"
          bind:value={scheduleWindowHours}
          required
        />
      </div>
    </Label>

    <div class="flex justify-end gap-4 pt-4">
      <Button type="button" onclick={() => open = false}>
        Annuler
      </Button>
      <Button type="submit" class="bg-option-500">
        Créer
      </Button>
    </div>
  </form>
</Modal>
