<script lang="ts">
	import { Button, Dropdown, DropdownItem , Label, Textarea, Input } from "flowbite-svelte";
    import { createConfig, updateConfig, deleteConfig } from '$lib/index';
    import { authStore, configs} from '$lib/store'
	import { ChevronDownOutline } from "flowbite-svelte-icons";
	import { onMount } from "svelte";
    import type { Config } from "$lib/type";


    let config_name: string = "Configurations";
    let textspacedisplay: boolean = false;
    let text: string = "";
    let newconfigname: string = "";
    let configlist: Config[] = [];
    let token: string | null = null;
    

    $: configlist = $configs;
    $: token = $authStore?.token ?? null;



    const handleClickDropdown = async (e: Event) => {
        e.preventDefault();
        const target = e.target as HTMLButtonElement;
        config_name = target.name;
        text = configlist.find(c => c.name === target.name)?.data || "";
        textspacedisplay = true;
        newconfigname = target.name;
    }

    const handleNewConfig = async (e: Event) => {
        e.preventDefault();
        config_name = "Configurations";
        text = "";
        textspacedisplay = true;
        newconfigname = "";
    }
    
    onMount(async () => {
        if (!token) {
            // Rediriger vers la page de connexion si le token n'existe pas
            window.location.href = '/';
        }
    });

    async function handlecreateConfig() {
        // Logique pour créer une nouvelle configuration
        console.log("Creating new configuration:", newconfigname, text);
        await createConfig($authStore?.email ?? "",newconfigname, text);
        config_name = newconfigname;
    }

    async function handleupdateConfig() {
        console.log("Updating configuration:", newconfigname, text);
        await updateConfig($authStore?.email ?? "", newconfigname, text);
    }

    async function handledeleteConfig() {
        console.log("Deleting configuration:", config_name);
        await deleteConfig($authStore?.email ?? "", newconfigname);
        config_name = "Configurations";
        text = "";
        textspacedisplay = false;
        newconfigname = "";
    }

</script>

<Button size="md" class="w-48 h-12">
    {config_name} <ChevronDownOutline class="ms-2 h-6 text-white" />
</Button>
<Dropdown simple isOpen={false} class="mt-2">
    {#each configlist as config}
        <DropdownItem name={config.name} onclick={handleClickDropdown}>{config.name}</DropdownItem>
    {/each}
</Dropdown>

<Button size="md" class="w-48 h-12 mt-4" onclick={handleNewConfig}>
    Create a new configuration
</Button>

{#if textspacedisplay}
    <Label for="textarea-id" class="mb-2">Votre script de configuration</Label>
    <Textarea id="textarea-id" placeholder="#!/bin/bash" rows={25} bind:value={text} class="w-full"/>
    <Label for="config-name" class="mb-2 mt-2">Nom de la configuration</Label>
    <Input id="config-name" type="text" placeholder="Configuration Name" class="mt-2 mb-2" bind:value={newconfigname} />
        <Button size="md" class="w-48 h-12 mt-2" onclick={handleupdateConfig}>Update Configuration</Button>
        <Button size="md" class="w-48 h-12 mt-2" onclick={handledeleteConfig}>Delete Configuration</Button>
        <Button size="md" class="w-48 h-12 mt-2" onclick={handlecreateConfig}>Save Configuration</Button>
        <Button size="md" class="w-48 h-12 mt-2" onclick={handledeleteConfig} disabled >Delete Configuration</Button>
{/if}