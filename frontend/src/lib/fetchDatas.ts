import { get } from 'svelte/store';
import { authStore } from './stores/authStore';


export interface ImageOption {
  value: string;
  name: string;
  status: string;
  Mindisk: number;
  Minram: number;
}

export type ImageGroupe = Record<string, ImageOption[]>;
export interface GroupedImageOption {
  group: string;   
  value: string;   
  name: string;
  status: string;
  Mindisk: number;
  Minram: number;
}

export interface GroupeImageName {
  name: string;
  value: string;
}

export interface FlavorOption {
  value: string;
  name: string;
  disk: number;
  ram: number;
  vcpus: number;
  rxtx_factor: number;
}

export interface NetworkOption {
  value: string;
  name: string;
}


export async function fetchAllFlavors(): Promise<FlavorOption[]> {
  const token = get(authStore);
  try {
    const res = await fetch('http://localhost:8080/serverpool/flavor', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les flavors");
    }
    const data = await res.json();
    return data.map((flavor: any) => ({
      value: flavor.id,
      name: flavor.name || flavor.id,
      disk: flavor.disk,
      ram: flavor.ram,
      vcpus: flavor.vcpus,
      rxtx_factor: flavor.rxtx_factor
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

export async function fetchAllNetworks(): Promise<NetworkOption[]> {
  const token = get(authStore);
  try {
    const res = await fetch('http://localhost:8080/serverpool/networks', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
      throw new Error("Impossible de récupérer les réseaux");
    }
    const data = await res.json();
    return data.map((net: any) => ({
      value: net.id,
      name: net.name || net.id,
      disk: net.disk,
      ram: net.ram,
      vcpus: net.vcpus,
      rxtx_factor: net.rxtx_factor
    }));
  } catch (err) {
    console.error(err);
    return [];
  }
}

export async function fetchGroupImages(group :string): Promise<ImageOption[]> {
  const token = get(authStore);

  try {
    const res = await fetch('http://localhost:8080/serverpool/imagegroup', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({group})
    });

    if (!res.ok) {
      throw new Error("Impossible de récupérer les images du groupe");
    }

    const data = await res.json();
    return data.map((img: any) => ({
      value: img.id,
      name: img.name || img.id,
      status: img.status,
      Mindisk: img.min_disk,
      Minram: img.min_ram
    }));
  } catch (err){
    console.error(err);
    return [];
  }
}

export async function fetchGroupImageName(): Promise<GroupeImageName[]> {
  const token = get(authStore);

  try {
    const res = await fetch(`http://localhost:8080/serverpool/groupimagesname`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`
      },
    });

    if (!res.ok) {
      throw new Error("Impossible de récupérer les images groupées");
    }

    const data: GroupeImageName[] = await res.json();
    return data;
  } catch (err) {
    console.error(err);
    return [];
  }
}