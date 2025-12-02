<script lang="ts">
    import type { Writable } from "svelte/store";

    export let title: string;
    export let store: Writable<any[]>;
    export let key: string;
</script>

<h2>{title}</h2>

<ul>
    {#each $store as item}
        <li>{item[key]}</li>
    {/each}
</ul>
