import { writable } from "svelte/store";

import {
    getAllImages,
    getAllFlavors,
    getAllNetworks,
    getAllServers,
    getAllServerPools,
    getAllConfigs,
} from "$lib/index";

import type {
    Image,
    Flavor,
    Network,
    Server,
    ServerPool,
    Config,
} from "../grpc/frontcontrol_pb";


// ==========================================================================
// Stores
// ==========================================================================
export const images = writable<Image[]>([]);
export const flavors = writable<Flavor[]>([]);
export const networks = writable<Network[]>([]);
export const servers = writable<Server[]>([]);
export const serverPools = writable<ServerPool[]>([]);
export const configs = writable<Config[]>([]);


// ==========================================================================
// Loaders (chargent les données et mettent à jour les stores)
// ==========================================================================

export async function loadImages(user: string) {
    const data = await getAllImages(user);
    images.set(data);
}

export async function loadFlavors(user: string) {
    const data = await getAllFlavors(user);
    flavors.set(data);
}

export async function loadNetworks(user: string) {
    const data = await getAllNetworks(user);
    networks.set(data);
}

export async function loadServers(user: string) {
    const data = await getAllServers(user);
    servers.set(data);
}

export async function loadServerPools(user: string) {
    const data = await getAllServerPools(user);
    serverPools.set(data);
}

export async function loadConfigs(user: string) {
    const data = await getAllConfigs(user);
    configs.set(data);
}


// ==========================================================================
// Helper pour tout charger d'un coup (infrastructure générale)
// ==========================================================================
export async function loadAll(user: string) {
    await Promise.all([
        loadImages(user),
        loadFlavors(user),
        loadNetworks(user),
        loadServers(user),
        loadServerPools(user),
        loadConfigs(user),
    ]);
}

export function resetAll() {
    images.set([]);
    flavors.set([]);
    networks.set([]);
    servers.set([]);
    serverPools.set([]);
    configs.set([]);
}