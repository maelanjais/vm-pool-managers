<script lang="ts">
  import { returnPoolsWithKey, attribVMinPool } from "$lib/grpc/attribVMService/attribVMService";

  let sshkey = $state("");
  let availablePools: { pool_id: string; user_id: string }[] = $state([]);
  let selectedPool: { pool_id: string; user_id: string } | null = $state(null);
  let vmIp = $state("");
  let vmUser = $state("");
  let vmAppPort = $state(0);
  let guacUrl = $state("");
  let loading = $state(false);
  let errorMsg = $state("");
  let noCoursFound = $state(false);
  let copied = $state(false);
  let appReady = $state(false);
  let probing = $state(false);
  let probeInterval: ReturnType<typeof setInterval> | null = null;

  function startProbing(ip: string, port: number) {
    appReady = false;
    probing = true;
    probeInterval = setInterval(async () => {
      try {
        const res = await fetch(`/api/app-status?ip=${encodeURIComponent(ip)}&port=${port}`);
        const data = await res.json();
        if (data.ready) {
          appReady = true;
          probing = false;
          if (probeInterval) { clearInterval(probeInterval); probeInterval = null; }
        }
      } catch { /* keep trying */ }
    }, 3000);
  }

  function fallbackCopy(text: string) {
    const el = document.createElement('textarea');
    el.value = text;
    el.style.position = 'fixed';
    el.style.opacity = '0';
    document.body.appendChild(el);
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
  }

  function copyCmd() {
    const text = `ssh ${vmUser}@${vmIp}`;
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text).catch(() => fallbackCopy(text));
    } else {
      fallbackCopy(text);
    }
    copied = true;
    setTimeout(() => copied = false, 2000);
  }

  async function handleSSHKey() {
    if (!sshkey.trim()) return;
    loading = true; errorMsg = ""; noCoursFound = false; availablePools = []; selectedPool = null; vmIp = "";
    try {
      availablePools = await returnPoolsWithKey(sshkey);
      if (availablePools.length === 0) noCoursFound = true;
    } catch { errorMsg = "Erreur lors de la récupération des cours disponibles."; }
    finally { loading = false; }
  }

  function computeUsername(poolId: string): string {
    let name = ("student_" + poolId).split("@")[0].toLowerCase();
    name = name.replace(/[^a-z0-9_.-]/g, "");
    if (name.length > 32) name = name.substring(0, 32);
    return name;
  }

  async function assignVM(pool: { pool_id: string; user_id: string }) {
    selectedPool = pool; loading = true; errorMsg = ""; vmIp = ""; vmUser = ""; vmAppPort = 0; guacUrl = "";
    appReady = false; probing = false;
    if (probeInterval) { clearInterval(probeInterval); probeInterval = null; }
    try {
      const result = await attribVMinPool(pool.pool_id, pool.user_id, sshkey);
      vmIp = result.ip;
      vmUser = result.username || computeUsername(pool.pool_id);
      vmAppPort = result.appPort ?? 0;
      fetch(`/api/guac-url?ip=${encodeURIComponent(result.ip)}`)
        .then(r => r.json())
        .then(data => { if (data.url) guacUrl = data.url; })
        .catch(() => {});
      if (vmAppPort > 0) startProbing(result.ip, vmAppPort);
    } catch (err: any) {
      errorMsg = err?.message || "Erreur lors de l'attribution de la VM.";
    } finally { loading = false; }
  }
</script>

<svelte:head>
  <title>CloudPoolManager — Portail Étudiant</title>
</svelte:head>

<div class="max-w-lg mx-auto py-10 animate-fade-up">

  {#if !vmIp}
    <div class="mb-8">
      <h1 class="text-3xl font-bold text-primary-800 mb-2" style="font-family: 'Source Sans 3', sans-serif; letter-spacing: -0.01em;">
        Portail étudiant
      </h1>
      <p class="text-sm text-neutral-500 leading-relaxed">
        Collez votre clé SSH publique pour accéder à votre machine virtuelle de travaux pratiques.
      </p>
    </div>

    <div class="card p-6 space-y-5">
      <div>
        <label for="sshkey" class="section-label mb-2 block">Clé publique SSH</label>
        <textarea
          id="sshkey"
          bind:value={sshkey}
          rows="4"
          placeholder="ssh-ed25519 AAAA..."
          class="field font-mono text-sm resize-none"
        ></textarea>
      </div>

      <button
        onclick={handleSSHKey}
        disabled={loading || !sshkey.trim()}
        class="btn btn-primary w-full"
      >
        {#if loading && !selectedPool}
          <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full" style="animation: spinnerGlow 0.6s linear infinite;"></span>
          Recherche en cours…
        {:else}
          Rechercher mes cours
        {/if}
      </button>

      {#if errorMsg}
        <div class="px-3 py-2.5 rounded bg-red-50 border border-red-200 text-red-700 text-sm animate-fade-in">{errorMsg}</div>
      {/if}
    </div>

    {#if noCoursFound}
      <div class="mt-6 card p-6 flex flex-col items-center text-center gap-3 animate-fade-in">
        <svg class="w-10 h-10 text-neutral-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
            d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
        </svg>
        <div>
          <p class="text-sm font-semibold text-neutral-700">Aucun cours lié à cette clé SSH</p>
          <p class="text-xs text-neutral-400 mt-1">Vérifiez que vous avez bien collé votre clé publique, ou contactez votre enseignant.</p>
        </div>
      </div>
    {/if}

    {#if availablePools.length > 0}
      <div class="mt-6">
        <p class="section-label mb-3 block">Cours disponibles</p>
        <div class="card overflow-hidden divide-y divide-neutral-100">
          {#each availablePools as pool}
            <div class="flex items-center justify-between px-5 py-3.5 hover:bg-neutral-50 transition-colors">
              <div>
                <p class="text-sm font-semibold text-neutral-900">{pool.pool_id}</p>
                <p class="text-xs text-neutral-500 mt-0.5">{pool.user_id}</p>
              </div>
              <button onclick={() => assignVM(pool)} disabled={loading} class="btn btn-primary text-xs px-4 py-2">
                {#if loading && selectedPool === pool}
                  <span class="w-3 h-3 border-2 border-white/30 border-t-white rounded-full" style="animation: spinnerGlow 0.6s linear infinite;"></span>
                  Attribution…
                {:else}
                  Rejoindre
                {/if}
              </button>
            </div>
          {/each}
        </div>
      </div>
    {/if}

  {:else}
    <div class="mb-8 animate-fade-in">
      <div class="flex items-center gap-3 mb-2">
        <span class="flex h-3 w-3 relative">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-60"></span>
          <span class="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
        </span>
        <h1 class="text-3xl font-bold text-primary-800" style="font-family: 'Source Sans 3', sans-serif;">VM attribuée</h1>
      </div>
      <p class="text-sm text-neutral-500 ml-6">
        {#if vmAppPort > 0 && !appReady}Démarrage en cours…{:else}Votre environnement est prêt.{/if}
      </p>
    </div>

    <div class="card p-6 space-y-5 animate-fade-in">

      {#if vmAppPort > 0}
        {#if appReady}
          <a
            href="http://{vmIp}:{vmAppPort}"
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center justify-center gap-2.5 w-full py-3.5 rounded-lg font-semibold text-base
              bg-amber-500 hover:bg-amber-400 text-white transition-all shadow-sm hover:shadow-md"
          >
            <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"/>
            </svg>
            Ouvrir l'application (port {vmAppPort})
          </a>
        {:else}
          <div class="flex items-center justify-center gap-2.5 w-full py-3.5 rounded-lg font-semibold text-base
            bg-neutral-200 text-neutral-500 cursor-not-allowed select-none">
            <span class="w-4 h-4 border-2 border-neutral-400/40 border-t-neutral-500 rounded-full shrink-0"
              style="animation: spinnerGlow 0.8s linear infinite;"></span>
            Démarrage de l'application…
          </div>
        {/if}
      {/if}

      {#if guacUrl}
        <a
          href={guacUrl}
          target="_blank"
          rel="noopener noreferrer"
          class="flex items-center justify-center gap-2.5 w-full py-3.5 rounded-lg font-semibold text-base
            bg-primary-700 hover:bg-primary-600 text-white transition-all shadow-sm hover:shadow-md"
        >
          <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
          </svg>
          Ouvrir le terminal web
        </a>
      {/if}

      {#if vmAppPort > 0 || guacUrl}
        <hr class="border-neutral-200"/>
      {/if}

      <div>
        <p class="section-label mb-2.5 block">Connexion SSH</p>
        <div class="flex items-center gap-2 bg-neutral-900 pl-4 pr-2 py-2 rounded-md font-mono">
          <svg class="w-4 h-4 text-primary-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3"/>
          </svg>
          <code class="text-sm text-green-400 select-all flex-1">ssh {vmUser}@{vmIp}</code>
          <button
            onclick={copyCmd}
            class="shrink-0 flex items-center gap-1.5 px-2.5 py-1.5 rounded text-xs font-semibold transition-all
              {copied ? 'bg-green-600 text-white' : 'bg-neutral-700 hover:bg-neutral-600 text-neutral-300'}"
            title="Copier"
          >
            {#if copied}
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
              </svg>
              Copié
            {:else}
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
              </svg>
              Copier
            {/if}
          </button>
        </div>
        <p class="text-xs text-neutral-400 mt-2">
          Si la connexion demande un mot de passe :
          <code class="font-mono text-neutral-500">ssh -i ~/.ssh/id_ed25519 {vmUser}@{vmIp}</code>
        </p>
      </div>

      <button
        onclick={() => {
        vmIp = ""; vmUser = ""; vmAppPort = 0; guacUrl = ""; availablePools = []; sshkey = "";
        appReady = false; probing = false;
        if (probeInterval) { clearInterval(probeInterval); probeInterval = null; }
      }}
        class="btn btn-secondary text-sm"
      >
        ← Retour
      </button>
    </div>
  {/if}

</div>
