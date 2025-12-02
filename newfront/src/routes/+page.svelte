<script lang="ts">
    import Section from "$lib/utils/section.svelte";
    import {
        images,
        flavors,
        networks,
        servers,
        serverPools,
        configs
    } from "$lib/store";

    import type {
        Image,
        Flavor,
        Network,
        Server,
        ServerPool,
        Config
    } from "$lib/grpc/frontcontrol_pb";

    type StoreSection<T> = {
        title: string;
        store: import("svelte/store").Writable<T[]>;
        key: string;
    };

    const sections: StoreSection<any>[] = [
        { title: "Images",       store: images,      key: "name" },
        { title: "Flavors",      store: flavors,     key: "name" },
        { title: "Networks",     store: networks,    key: "name" },
        { title: "Servers",      store: servers,     key: "name" },
        { title: "Server Pools", store: serverPools, key: "name" },
        { title: "Configs",      store: configs,     key: "name" }
    ];
</script>

{#each sections as section}
    <Section
        title={section.title}
        store={section.store}
        key={section.key}
    />
{/each}
