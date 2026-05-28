<script lang="ts">
  import type { Image, Flavor, Network, Config } from '$lib/type';

  let {
    open = $bindable(),
    images,
    flavors,
    networks,
    configs,
    selectedGroupImage = $bindable(),
    selectedImage = $bindable(),
    selectedFlavor = $bindable(),
    selectedNetwork = $bindable(),
    selectedConfigFile = $bindable(),
    scheduleDay = $bindable(),
    scheduleTime = $bindable(),
    scheduleWindowHours = $bindable(),
    offDays = $bindable(),
    appPort = $bindable(),
    createError,
    createSuccess,
    handleCreateServerpool,
    getUniqueFirstAlphaBlocks,
    filterImagesByPrefix,
  }: {
    open: boolean;
    images: Image[];
    flavors: Flavor[];
    networks: Network[];
    configs: Config[];
    selectedGroupImage: string | null;
    selectedImage: string | null;
    selectedFlavor: string;
    selectedNetwork: string;
    selectedConfigFile: string;
    scheduleDay: string;
    scheduleTime: string;
    scheduleWindowHours: number | undefined;
    offDays: { monday:boolean; tuesday:boolean; wednesday:boolean; thursday:boolean; friday:boolean; saturday:boolean; sunday:boolean; };
    appPort: number;
    createError: string;
    createSuccess: boolean;
    handleCreateServerpool: (e: Event) => void;
    getUniqueFirstAlphaBlocks: (images: Image[]) => string[];
    filterImagesByPrefix: (images: Image[], prefix: string) => Image[];
  } = $props();

  // Maps snapshot suffix → human label
  const jupyterSnapshotLabels: Record<string, string> = {
    'scipy':       'Python scientifique (scipy-notebook)',
    'scipy-plus':  'Python scientifique+',
    'datascience': 'Data Science (Python + R + Julia)',
    'julia':       'Julia',
    'bio583':      'BIO583',
    'eco589':      'ECO589',
    'compeco':     'Computational Economics',
    'mec431':      'MEC431',
    'mec558':      'MEC558',
    'map579':      'MAP579',
    'mec552a':     'MEC552A',
    'mec552b':     'MEC552B',
    'mec568':      'MEC568',
    'mec581':      'MEC581',
    'mec666':      'MEC666',
  };

  function getJupyterSnapshots(): { id: string; label: string }[] {
    return images
      .filter(i => i.name.startsWith('jupyter-snapshot-'))
      .map(i => {
        const suffix = i.name.replace('jupyter-snapshot-', '');
        return { id: i.id, label: jupyterSnapshotLabels[suffix] ?? suffix };
      })
      .sort((a, b) => a.label.localeCompare(b.label));
  }

  const JUPYTER_GROUP = 'JupyterHub';

  function getImageGroups(): string[] {
    // Exclude jupyter-snapshot-* and any jupyterhub* images from regular groups
    const regular = images.filter(i =>
      !i.name.startsWith('jupyter-snapshot-') &&
      !i.name.toLowerCase().startsWith('jupyterhub')
    );
    const groups = getUniqueFirstAlphaBlocks(regular);
    if (getJupyterSnapshots().length > 0) {
      return [JUPYTER_GROUP, ...groups];
    }
    return groups;
  }

  function onGroupChange(group: string) {
    selectedGroupImage = group;
    selectedImage = null;
    appPort = 0;
    selectedConfigFile = '';
  }

  function onJupyterSnapshotChange(imgId: string) {
    selectedImage = imgId;
    appPort = 8888;
    // Auto-select the matching autostart config (jupyter-snapshot-{suffix})
    const img = images.find(i => i.id === imgId);
    if (img) {
      const suffix = img.name.replace('jupyter-snapshot-', '');
      selectedConfigFile = `jupyter-snapshot-${suffix}`;
    }
  }

  const offDayLabels: { key: keyof typeof offDays; label: string }[] = [
    { key: 'monday', label: 'Lun' }, { key: 'tuesday', label: 'Mar' },
    { key: 'wednesday', label: 'Mer' }, { key: 'thursday', label: 'Jeu' },
    { key: 'friday', label: 'Ven' }, { key: 'saturday', label: 'Sam' },
    { key: 'sunday', label: 'Dim' },
  ];

  function getImageDiskGb(img: Image): number {
    if (img.minDiskGigabytes > 0) return img.minDiskGigabytes;
    if (img.sizeBytes > 0n) return Math.ceil(Number(img.sizeBytes) / (1024 ** 3));
    return 0;
  }

  // Recommended = the 2 smallest vd flavors with disk >= needed
  function getRecommendedFlavorIds(): Set<string> {
    if (!selectedImage) return new Set();
    const img = images.find(i => i.id === selectedImage);
    if (!img) return new Set();
    const needed = getImageDiskGb(img);
    if (needed === 0) return new Set();
    const top2 = flavors
      .filter(f => f.name.toLowerCase().startsWith('vd') && f.disk >= needed)
      .sort((a, b) => a.disk - b.disk || a.vcpus - b.vcpus)
      .slice(0, 2);
    return new Set(top2.map(f => f.id));
  }

  function flavorStatus(f: Flavor): 'recommended' | 'ok' | 'incompatible' | 'unknown' {
    if (!selectedImage) return 'unknown';
    const img = images.find(i => i.id === selectedImage);
    if (!img) return 'unknown';
    const needed = getImageDiskGb(img);
    if (needed === 0) return 'unknown';
    if (f.disk < needed) return 'incompatible';
    if (getRecommendedFlavorIds().has(f.id)) return 'recommended';
    return 'ok';
  }

  function formatRam(ram: number): string {
    if (ram <= 0) return '—';
    // OpenStack returns RAM in MB; if value < 16 it was likely already converted to GB
    if (ram < 16) return `${ram} GB`;
    if (ram >= 1024) return `${Math.round(ram / 1024)} GB`;
    return `${ram} MB`;
  }
</script>

{#if open}
  <div class="modal-overlay" role="dialog" aria-modal="true">
    <div class="modal-box modal-box-lg" style="max-height:90vh;overflow-y:auto;">

      <div class="flex items-center justify-between mb-6 pb-5 border-b border-neutral-200">
        <div>
          <h3 class="text-lg font-bold text-neutral-900" style="font-family: 'Source Sans 3', sans-serif;">Nouveau Serverpool</h3>
          <p class="text-sm text-neutral-500 mt-0.5">Configurez un groupe de VMs pour vos étudiants</p>
        </div>
        <button onclick={() => open = false} class="text-neutral-400 hover:text-neutral-700 transition-colors p-1 rounded hover:bg-neutral-100">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      {#if createSuccess}
        <div class="flex flex-col items-center justify-center py-16 gap-5 animate-fade-in">
          <div class="w-14 h-14 rounded-full bg-green-50 border border-green-200 flex items-center justify-center">
            <svg class="w-7 h-7 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
            </svg>
          </div>
          <div class="text-center">
            <p class="text-base font-bold text-neutral-900">Serverpool créé avec succès</p>
            <p class="text-sm text-neutral-500 mt-1">Les VMs vont démarrer dans quelques instants.</p>
          </div>
          <button onclick={() => open = false} class="btn btn-primary px-8">OK</button>
        </div>
      {:else}

      {#if createError}
        <div class="mb-5 px-4 py-3 rounded bg-red-50 border border-red-200 text-red-700 text-sm animate-fade-in">{createError}</div>
      {/if}

      <form class="space-y-6" onsubmit={handleCreateServerpool}>

        <!-- Section 1 + 2 -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">

          <!-- 1. Général -->
          <div class="card-elevated p-5 space-y-4">
            <h4 class="text-xs font-bold text-primary-700 uppercase tracking-widest border-b border-neutral-200 pb-2">1. Général</h4>

            <div class="space-y-1.5">
              <label class="section-label">Nom du serverpool</label>
              <input class="field" type="text" name="namesp" placeholder="TP-Reseaux-2026" required />
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div class="space-y-1.5">
                <label class="section-label">Min VMs</label>
                <input class="field" type="number" name="min_vm" min="1" value="1" required />
              </div>
              <div class="space-y-1.5">
                <label class="section-label">Max VMs</label>
                <input class="field" type="number" name="max_vm" min="1" value="5" required />
              </div>
            </div>

            <div class="space-y-1.5">
              <label class="section-label" for="app_port">Port application <span class="text-neutral-400 font-normal">(optionnel)</span></label>
              <div class="flex items-center gap-2">
                <input
                  id="app_port"
                  class="field w-36"
                  type="number"
                  min="1" max="65535"
                  bind:value={appPort}
                  placeholder="ex: 8888"
                />
                <p class="text-xs text-neutral-400 leading-snug">
                  Si l'image expose une app web (Jupyter = 8888), les étudiants verront un bouton d'accès direct.
                </p>
              </div>
            </div>
          </div>

          <!-- 2. Infrastructure — OS + Réseau -->
          <div class="card-elevated p-5 space-y-4">
            <h4 class="text-xs font-bold text-primary-700 uppercase tracking-widest border-b border-neutral-200 pb-2">2. Infrastructure</h4>

            <div class="space-y-1.5">
              <label class="section-label">Système d'exploitation</label>
              <select class="field" value={selectedGroupImage ?? ''} onchange={(e) => onGroupChange((e.target as HTMLSelectElement).value)} required>
                <option disabled selected value="">Famille d'OS…</option>
                {#each getImageGroups() as group}
                  <option value={group}>{group}</option>
                {/each}
              </select>

              {#if selectedGroupImage === JUPYTER_GROUP}
                <select class="field mt-2" value={selectedImage ?? ''} onchange={(e) => onJupyterSnapshotChange((e.target as HTMLSelectElement).value)} required>
                  <option disabled selected value="">Environnement Jupyter…</option>
                  {#each getJupyterSnapshots() as snap}
                    <option value={snap.id}>{snap.label}</option>
                  {/each}
                </select>
                <p class="text-xs text-neutral-400">Port 8888 activé automatiquement · image Docker pré-installée</p>
              {:else if selectedGroupImage}
                <select class="field mt-2" bind:value={selectedImage} required>
                  <option disabled selected value="">Version exacte…</option>
                  {#each filterImagesByPrefix(images.filter(i => !i.name.startsWith('jupyter-snapshot-') && !i.name.toLowerCase().startsWith('jupyterhub')), selectedGroupImage) as img}
                    <option value={img.id}>{img.name}{img.minDiskGigabytes > 0 ? ` (${img.minDiskGigabytes} GB requis)` : ''}</option>
                  {/each}
                </select>
              {/if}
            </div>

            <div class="space-y-1.5">
              <label class="section-label">Réseau</label>
              <select class="field" bind:value={selectedNetwork} required>
                <option disabled selected value="">Choisir…</option>
                {#each networks as n}
                  <option value={n.id}>{n.name}</option>
                {/each}
              </select>
            </div>
          </div>
        </div>

        <!-- Flavor — pleine largeur -->
        <div class="card-elevated p-5 space-y-3">
          <h4 class="text-xs font-bold text-primary-700 uppercase tracking-widest border-b border-neutral-200 pb-2">
            Flavor
            {#if selectedImage}
              {@const img = images.find(i => i.id === selectedImage)}
              {#if img}
                {@const needed = getImageDiskGb(img)}
                {#if needed > 0}
                  <span class="text-neutral-400 font-normal normal-case tracking-normal ml-2">— image {img.name.split('-')[0]}… · {needed} GB requis</span>
                {/if}
              {/if}
            {/if}
          </h4>

          {#if !selectedImage}
            <select class="field" bind:value={selectedFlavor} required>
              <option disabled selected value="">Sélectionnez d'abord une image…</option>
              {#each flavors as f}
                <option value={f.id}>{f.name} — {f.disk} GB · {f.vcpus} vCPU · {formatRam(f.ram)}</option>
              {/each}
            </select>
          {:else}
            {@const vdFlavors = flavors
              .filter(f => f.name.toLowerCase().startsWith('vd'))
              .sort((a, b) => {
                const rank = { recommended: 0, ok: 1, unknown: 2, incompatible: 3 };
                return rank[flavorStatus(a)] - rank[flavorStatus(b)] || a.disk - b.disk || a.vcpus - b.vcpus;
              })}
            {@const otherFlavors = flavors
              .filter(f => !f.name.toLowerCase().startsWith('vd'))
              .sort((a, b) => {
                const rank = { recommended: 0, ok: 1, unknown: 2, incompatible: 3 };
                return rank[flavorStatus(a)] - rank[flavorStatus(b)] || a.name.localeCompare(b.name, undefined, {numeric:true});
              })}

            <div class="grid grid-cols-1 gap-3">
              <!-- vd flavors -->
              {#if vdFlavors.length > 0}
                <div>
                  <p class="section-label mb-2">Flavors vd</p>
                  <div class="border border-neutral-200 rounded overflow-hidden divide-y divide-neutral-100">
                    {#each vdFlavors as f}
                      {@const status = flavorStatus(f)}
                      <button
                        type="button"
                        onclick={() => status !== 'incompatible' && (selectedFlavor = f.id)}
                        class="w-full text-left px-4 py-2.5 flex items-center gap-4 transition-colors
                          {selectedFlavor === f.id
                            ? 'bg-primary-50'
                            : status === 'incompatible'
                              ? 'bg-neutral-50 cursor-not-allowed'
                              : 'hover:bg-neutral-50 cursor-pointer'}"
                      >
                        <!-- Selected indicator -->
                        <span class="w-3.5 h-3.5 rounded-full border-2 shrink-0 flex items-center justify-center
                          {selectedFlavor === f.id ? 'border-primary-700 bg-primary-700' : 'border-neutral-300'}">
                          {#if selectedFlavor === f.id}
                            <span class="w-1.5 h-1.5 rounded-full bg-white"></span>
                          {/if}
                        </span>

                        <!-- Name -->
                        <span class="text-sm font-bold w-16 shrink-0 {status === 'incompatible' ? 'text-neutral-400' : 'text-neutral-900'}">{f.name}</span>

                        <!-- Specs -->
                        <span class="text-xs text-neutral-500 flex items-center gap-3">
                          <span title="Disque">{f.disk} GB</span>
                          <span class="text-neutral-300">·</span>
                          <span title="CPU">{f.vcpus} vCPU</span>
                          <span class="text-neutral-300">·</span>
                          <span title="RAM">{formatRam(f.ram)}</span>
                        </span>

                        <!-- Badge -->
                        <span class="ml-auto text-xs font-bold shrink-0
                          {status === 'recommended' ? 'text-green-700 bg-green-50 border border-green-200 px-2 py-0.5 rounded'
                          : status === 'incompatible' ? 'text-red-500'
                          : ''}">
                          {#if status === 'recommended'}★ Recommandé
                          {:else if status === 'incompatible'}✗ Disque insuffisant ({f.disk} GB &lt; {getImageDiskGb(images.find(i => i.id === selectedImage)!)} GB)
                          {/if}
                        </span>
                      </button>
                    {/each}
                  </div>
                </div>
              {/if}

              <!-- Autres flavors (repliées par défaut) -->
              {#if otherFlavors.length > 0}
                <details class="group">
                  <summary class="section-label cursor-pointer select-none hover:text-neutral-600 list-none flex items-center gap-1">
                    <svg class="w-3 h-3 transition-transform group-open:rotate-90" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                    </svg>
                    Autres flavors ({otherFlavors.length})
                  </summary>
                  <div class="mt-2 border border-neutral-200 rounded overflow-hidden divide-y divide-neutral-100">
                    {#each otherFlavors as f}
                      {@const status = flavorStatus(f)}
                      <button
                        type="button"
                        onclick={() => status !== 'incompatible' && (selectedFlavor = f.id)}
                        class="w-full text-left px-4 py-2 flex items-center gap-4 transition-colors
                          {selectedFlavor === f.id
                            ? 'bg-primary-50'
                            : status === 'incompatible'
                              ? 'bg-neutral-50 cursor-not-allowed'
                              : 'hover:bg-neutral-50 cursor-pointer'}"
                      >
                        <span class="w-3.5 h-3.5 rounded-full border-2 shrink-0
                          {selectedFlavor === f.id ? 'border-primary-700 bg-primary-700' : 'border-neutral-300'}"></span>
                        <span class="text-sm font-semibold w-32 shrink-0 truncate {status === 'incompatible' ? 'text-neutral-400' : 'text-neutral-800'}">{f.name}</span>
                        <span class="text-xs text-neutral-400">{f.disk} GB · {f.vcpus} vCPU · {formatRam(f.ram)}</span>
                        {#if status === 'incompatible'}
                          <span class="ml-auto text-xs text-red-400 shrink-0">✗ Disque insuffisant</span>
                        {/if}
                      </button>
                    {/each}
                  </div>
                </details>
              {/if}
            </div>

            {#if selectedFlavor}
              {@const sel = flavors.find(f => f.id === selectedFlavor)}
              {#if sel}
                <p class="text-xs text-neutral-500">
                  Sélectionné : <span class="font-bold text-primary-700">{sel.name}</span>
                  <span class="text-neutral-400 ml-1">— {sel.disk} GB · {sel.vcpus} vCPU · {formatRam(sel.ram)}</span>
                </p>
              {/if}
            {/if}
          {/if}
        </div>

        <!-- Section 3: Options avancées -->
        <div class="card-elevated p-5 space-y-5">
          <h4 class="text-xs font-bold text-primary-700 uppercase tracking-widest border-b border-neutral-200 pb-2">3. Options avancées</h4>

          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">

            <!-- Left: Config + off days -->
            <div class="space-y-4">
              <div class="space-y-1.5">
                <label class="section-label">Script d'initialisation</label>
                <select class="field" bind:value={selectedConfigFile}>
                  <option value="">Aucun (défaut)</option>
                  {#each configs as c}
                    <option value={c.name}>{c.name}</option>
                  {/each}
                </select>
              </div>

              <div>
                <p class="section-label mb-2.5 block">Jours de fermeture</p>
                <div class="flex flex-wrap gap-2">
                  {#each offDayLabels as { key, label }}
                    <button
                      type="button"
                      class="w-9 h-9 rounded text-xs font-bold transition-all
                        {offDays[key]
                          ? 'bg-primary-700 text-white border border-primary-800'
                          : 'bg-white text-neutral-500 border border-neutral-300 hover:border-primary-400 hover:text-primary-600'}"
                      onclick={() => offDays[key] = !offDays[key]}
                    >{label}</button>
                  {/each}
                </div>
                <p class="text-xs text-neutral-400 mt-2">Suspend les VMs ces jours pour économiser les ressources</p>
              </div>
            </div>

            <!-- Right: Schedule -->
            <div class="space-y-3">
              <p class="section-label block">Planning de démarrage</p>
              <div class="grid grid-cols-3 gap-3">
                <div class="space-y-1.5">
                  <label class="section-label">Jour</label>
                  <select class="field" bind:value={scheduleDay}>
                    <option value="">Aucun</option>
                    <option value="1">Lundi</option>
                    <option value="2">Mardi</option>
                    <option value="3">Mercredi</option>
                    <option value="4">Jeudi</option>
                    <option value="5">Vendredi</option>
                  </select>
                </div>
                <div class="space-y-1.5">
                  <label class="section-label">Heure</label>
                  <input class="field" type="time" bind:value={scheduleTime} />
                </div>
                <div class="space-y-1.5">
                  <label class="section-label">Durée (h)</label>
                  <input class="field" type="number" min="1" max="24" bind:value={scheduleWindowHours} placeholder="4" />
                </div>
              </div>
              <p class="text-xs text-neutral-400">Laissez vide pour démarrer manuellement</p>
            </div>

          </div>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-end gap-3 pt-1">
          <button type="button" onclick={() => open = false} class="btn btn-secondary text-sm">
            Annuler
          </button>
          <button type="submit" class="btn btn-primary text-sm px-6">
            Créer le serverpool
          </button>
        </div>

      </form>
      {/if}
    </div>
  </div>
{/if}
