<script lang="ts">
  import { onMount } from 'svelte';
  import { authStore } from '$lib/store';
  import { simpleMode, refreshInterval } from '$lib/store/uiStore';
  import { browser } from '$app/environment';

  interface VMInstance {
    id: string; name: string; ip: string; public_ip: string; az: string;
    status: string; healthy: boolean; activity_status: string;
    registered_at: string; last_seen: string; raw_meta: Record<string, string>;
    guac_url?: string;
    student?: string;        // étudiant attribué (par IP)
    is_instructor?: boolean; // VM de l'enseignant
  }
  interface InventoryPool { pool_id: string; user_id: string; vms: VMInstance[]; }

  let pools: InventoryPool[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let lastRefresh = $state('');
  let refreshing = $state(false);

  async function fetchInventory(silent = false) {
    if (!silent) loading = true; else refreshing = true;
    try {
      const res = await fetch('/api/inventory');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      pools = await res.json();
      lastRefresh = new Date().toLocaleTimeString('fr-FR');
      error = '';
    } catch { error = "Impossible de charger l'inventaire"; }
    finally { loading = false; refreshing = false; }
  }

  onMount(() => {
    if (!browser) return;
    if (!$authStore || $authStore.role !== 'admin') { window.location.href = '/'; return; }
    fetchInventory();
  });

  // Auto-refresh : intervalle configurable (Paramètres). Se recrée si l'intervalle change.
  $effect(() => {
    if (!browser || !$authStore || $authStore.role !== 'admin') return;
    const ms = Math.max(3, $refreshInterval || 15) * 1000;
    const id = setInterval(() => fetchInventory(true), ms);
    return () => clearInterval(id);
  });

  function timeSince(dateStr: string): string {
    const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
    if (diff < 60) return `${diff}s`;
    if (diff < 3600) return `${Math.floor(diff/60)}min`;
    if (diff < 86400) return `${Math.floor(diff/3600)}h`;
    return `${Math.floor(diff/86400)}j`;
  }

  const totalVMs = $derived(pools.reduce((a, p) => a + p.vms.length, 0));
  const healthyVMs = $derived(pools.reduce((a, p) => a + p.vms.filter(v => v.healthy).length, 0));
  const readyVMs = $derived(pools.reduce((a, p) => a + p.vms.filter(v => v.status === 'ready').length, 0));
  const activeVMs = $derived(pools.reduce((a, p) => a + p.vms.filter(v => v.activity_status !== 'idle').length, 0));
</script>

<svelte:head><title>Inventaire VM — CloudPoolManager</title></svelte:head>

{#if $simpleMode}
<div class="space-y-6 animate-fade-up">
  <div class="flex items-start justify-between">
    <div>
      <h1 class="text-3xl font-bold text-primary-800" style="font-family: 'Source Sans 3', sans-serif;">Mes étudiants</h1>
      <p class="text-sm text-neutral-500 mt-1">Suivez la connexion de vos étudiants en temps réel</p>
    </div>
    <button onclick={() => fetchInventory(true)} disabled={refreshing} class="btn btn-secondary text-xs px-3.5 py-2">
      <svg class="w-3.5 h-3.5 {refreshing ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
      </svg>
      Actualiser
    </button>
  </div>

  {#if loading}
    <div class="flex justify-center py-20"><div class="w-8 h-8 rounded-full border-2 border-neutral-200 border-t-primary-700" style="animation: spinnerGlow 0.7s linear infinite;"></div></div>
  {:else if error}
    <div class="card px-4 py-3 border-red-200 bg-red-50 text-red-700 text-sm">{error}</div>
  {:else if pools.length === 0}
    <div class="card flex flex-col items-center justify-center py-20 text-center">
      <p class="text-neutral-500 text-sm">Aucun cours actif pour le moment</p>
    </div>
  {:else}
    <div class="space-y-4">
      {#each pools as pool, pi}
        {@const activeVms = pool.vms.filter(v => v.activity_status !== 'idle')}
        {@const connectedStudents = pool.vms.filter(v => v.activity_status !== 'idle' && v.student)}
        {@const readyVms = pool.vms.filter(v => v.status === 'ready' && !v.is_instructor)}
        <div class="card overflow-hidden animate-fade-up" style="animation-delay:{pi*0.06}s">
          <div class="flex items-center justify-between px-5 py-4 border-b border-neutral-100">
            <div>
              <h2 class="text-sm font-bold text-neutral-900">{pool.pool_id}</h2>
              <p class="text-xs text-neutral-400 mt-0.5">
                <span class="{connectedStudents.length > 0 ? 'text-green-600 font-semibold' : 'text-neutral-400'}">
                  {connectedStudents.length} étudiant{connectedStudents.length > 1 ? 's' : ''} connecté{connectedStudents.length > 1 ? 's' : ''}
                </span>
                · {readyVms.length} machine{readyVms.length > 1 ? 's' : ''} disponible{readyVms.length > 1 ? 's' : ''}
              </p>
            </div>
            <div class="flex items-center gap-1.5">
              {#if activeVms.length > 0}
                <span class="animate-ping absolute inline-flex h-2 w-2 rounded-full bg-green-400 opacity-60"></span>
                <span class="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
                <span class="text-xs text-green-600 font-semibold">En cours</span>
              {:else}
                <span class="inline-flex rounded-full h-2 w-2 bg-neutral-300"></span>
                <span class="text-xs text-neutral-400">En attente</span>
              {/if}
            </div>
          </div>
          <div class="divide-y divide-neutral-50">
            {#each pool.vms as vm}
              {@const connected = vm.activity_status !== 'idle'}
              {@const label = vm.student ? vm.student : connected ? 'Connexion personnelle (enseignant)' : vm.is_instructor ? 'VM enseignant (réservée)' : vm.status === 'ready' ? 'Machine libre' : 'Démarrage…'}
              <div class="flex items-center justify-between gap-3 px-5 py-3 transition-colors {connected ? 'bg-green-50/70 dark:bg-green-900/10' : 'hover:bg-neutral-50 dark:hover:bg-white/[0.03]'}">
                <div class="flex items-center gap-3 min-w-0">
                  <!-- Avatar : initiale de l'étudiant, ou icône ; vert vif si connecté -->
                  <div class="relative w-9 h-9 rounded-full flex items-center justify-center text-sm font-bold shrink-0 transition-colors
                    {connected ? 'bg-green-500 text-white shadow-sm' : vm.is_instructor ? 'bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300' : 'bg-neutral-100 text-neutral-400 dark:bg-neutral-800'}">
                    {#if vm.student}
                      {vm.student.charAt(0).toUpperCase()}
                    {:else if connected || vm.is_instructor}
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/></svg>
                    {:else}
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
                    {/if}
                    {#if connected}
                      <span class="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full bg-green-500 ring-2 ring-white dark:ring-[#13151f]"></span>
                    {/if}
                  </div>
                  <div class="min-w-0">
                    <p class="text-sm font-semibold truncate {connected ? 'text-neutral-900 dark:text-white' : 'text-neutral-500 dark:text-neutral-400'}">{label}</p>
                    <p class="text-[11px] text-neutral-400 font-mono truncate">{vm.name}</p>
                  </div>
                </div>
                <div class="flex items-center gap-3 shrink-0">
                  {#if connected}
                    <span class="badge badge-ready">● En ligne</span>
                  {:else if vm.student}
                    <span class="text-xs text-neutral-400">Hors ligne</span>
                  {:else if vm.is_instructor}
                    <span class="text-xs text-neutral-400">Réservée</span>
                  {:else if vm.status === 'ready'}
                    <span class="text-xs text-neutral-400">En attente</span>
                  {:else}
                    <span class="text-xs text-amber-600">Démarrage…</span>
                  {/if}
                  {#if vm.guac_url}
                    <a href={vm.guac_url} target="_blank" rel="noopener" class="btn btn-secondary text-xs px-2 py-1">Terminal</a>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
{:else}
<div class="space-y-7 animate-fade-up">

  <!-- Header -->
  <div class="flex items-start justify-between">
    <div>
      <h1 class="text-3xl font-bold text-primary-800" style="font-family: 'Source Sans 3', sans-serif;">Inventaire</h1>
      <p class="text-sm text-neutral-500 mt-1">Supervision en temps réel des instances provisionnées</p>
    </div>
    <div class="flex items-center gap-3">
      {#if lastRefresh}
        <span class="text-xs text-neutral-400">Maj {lastRefresh}</span>
      {/if}
      <button
        onclick={() => fetchInventory(true)}
        disabled={refreshing}
        class="btn btn-secondary text-xs px-3.5 py-2 gap-1.5"
      >
        <svg class="w-3.5 h-3.5 {refreshing ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
        </svg>
        Actualiser
      </button>
    </div>
  </div>

  <!-- Stats -->
  {#if !loading && !error}
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      {#each [
        { label: 'Pools',       value: pools.length,                   accent: 'stat-accent-indigo',  color: 'text-primary-700' },
        { label: 'VMs total',   value: totalVMs,                       accent: 'stat-accent-violet',  color: 'text-primary-500' },
        { label: 'Santé',       value: `${healthyVMs}/${totalVMs}`,    accent: 'stat-accent-emerald', color: 'text-green-600'   },
        { label: 'Actives SSH', value: activeVMs,                      accent: 'stat-accent-amber',   color: 'text-amber-600'   },
      ] as stat, i}
        <div class="card card-interactive p-5 animate-fade-up" style="animation-delay:{i*0.05}s">
          <p class="section-label mb-2">{stat.label}</p>
          <p class="text-3xl font-bold {stat.color} tabular-nums tracking-tight">{stat.value}</p>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Loading -->
  {#if loading}
    <div class="flex flex-col items-center justify-center py-24 gap-4">
      <div class="w-9 h-9 rounded-full border-2 border-neutral-200 border-t-primary-700" style="animation: spinnerGlow 0.7s linear infinite;"></div>
      <p class="text-sm text-neutral-500">Chargement de l'inventaire…</p>
    </div>
  {/if}

  <!-- Error -->
  {#if error}
    <div class="card px-4 py-3 border-red-200 bg-red-50 text-red-700 text-sm animate-fade-in">{error}</div>
  {/if}

  <!-- Pool sections -->
  {#if !loading && !error}
    {#each pools as pool, pi}
      <div class="card overflow-hidden animate-fade-up" style="animation-delay:{pi*0.06}s">
        <!-- Pool header -->
        <div class="flex items-center justify-between px-5 py-3.5 bg-neutral-50 border-b border-neutral-200">
          <div class="flex items-center gap-3">
            <div class="relative flex h-2.5 w-2.5">
              {#if pool.vms.every(v => v.healthy)}
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-60"></span>
              {/if}
              <span class="relative inline-flex rounded-full h-2.5 w-2.5 {pool.vms.every(v => v.healthy) ? 'bg-green-500' : 'bg-red-500'}"></span>
            </div>
            <span class="text-sm font-bold text-neutral-900">{pool.pool_id}</span>
            <span class="text-xs text-neutral-500">{pool.user_id}</span>
          </div>
          <span class="text-xs text-neutral-400 tabular-nums">{pool.vms.length} VM{pool.vms.length > 1 ? 's' : ''}</span>
        </div>

        <!-- Table -->
        <div class="overflow-x-auto">
          <table class="data-table">
            <thead>
              <tr>
                <th>Nom</th>
                <th>IP</th>
                <th>Statut</th>
                <th>Santé</th>
                <th>Activité</th>
                <th>Terminal</th>
                <th class="text-right">Dernière activité</th>
              </tr>
            </thead>
            <tbody>
              {#each pool.vms as vm}
                <tr class="transition-colors">
                  <td>
                    <div class="flex flex-col gap-0.5">
                      <span class="font-mono text-xs text-neutral-700 dark:text-neutral-300">{vm.name}</span>
                      {#if vm.is_instructor}
                        <span class="text-[10px] font-semibold text-primary-600 dark:text-primary-400">● VM enseignant</span>
                      {:else if vm.student}
                        <span class="text-[10px] text-neutral-500">👤 {vm.student}</span>
                      {/if}
                    </div>
                  </td>
                  <td><span class="font-mono text-xs text-neutral-700">{vm.ip}</span></td>
                  <td>
                    <span class="badge {vm.status === 'ready' ? 'badge-ready' : vm.status === 'starting' ? 'badge-starting' : 'badge-error'}">
                      {vm.status}
                    </span>
                  </td>
                  <td>
                    <div class="flex items-center gap-1.5">
                      <span class="w-1.5 h-1.5 rounded-full {vm.healthy ? 'bg-green-500' : 'bg-red-500'}"></span>
                      <span class="text-xs font-medium {vm.healthy ? 'text-green-700' : 'text-red-700'}">{vm.healthy ? 'OK' : 'KO'}</span>
                    </div>
                  </td>
                  <td>
                    {#if vm.activity_status && vm.activity_status !== 'idle'}
                      <span class="badge badge-info gap-1.5">
                        <span class="relative flex h-1.5 w-1.5">
                          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
                          <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-sky-400"></span>
                        </span>
                        Sur Jupyter
                      </span>
                    {:else}
                      <span class="text-xs text-neutral-400">Inactif</span>
                    {/if}
                  </td>
                  <td>
                    {#if vm.guac_url}
                      <a href={vm.guac_url} target="_blank" rel="noopener"
                         class="btn btn-secondary text-xs px-2 py-1 flex items-center gap-1.5 w-fit">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                        </svg>
                        Terminal
                      </a>
                    {:else}
                      <span class="text-xs text-neutral-400">—</span>
                    {/if}
                  </td>
                  <td class="text-right">
                    <span class="text-xs text-neutral-400 tabular-nums">il y a {timeSince(vm.last_seen)}</span>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/each}

    {#if pools.length === 0}
      <div class="card flex flex-col items-center justify-center py-24 text-center">
        <svg class="w-10 h-10 text-neutral-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
            d="M5 12h14M12 5l7 7-7 7"/>
        </svg>
        <p class="text-neutral-500 text-sm font-medium">Aucune VM provisionnée pour le moment</p>
        <p class="text-neutral-400 text-xs mt-1">Les instances apparaîtront ici une fois démarrées</p>
      </div>
    {/if}
  {/if}
</div>
{/if}
