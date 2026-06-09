<script lang="ts">
  import { onMount } from 'svelte';
  import { authStore, serverPools } from '$lib/store';
  import { browser } from '$app/environment';
  import ConfirmModal from '$lib/components/ConfirmModal.svelte';

  interface Grade {
    student: string;
    score: number;
    max_score: number;
    status: string;
  }

  let allPools = $derived($serverPools as any[]);
  let selectedPool: { name: string; userId: string } | null = $state(null);
  let assignments: string[] = $state([]);
  let selectedAssignment = $state('');
  let grades: Grade[] = $state([]);
  let jupyterURL = $state('');      // proxy URL (for display)
  let jupyterDirectURL = $state(''); // direct VM URL (for iframe)
  let formgraderBaseURL = $state(''); // direct URL for Formgrader (new tab)

  let loadingAssignments = $state(false);
  let loadingGrades = $state(false);
  let releasing = $state(false);
  let collecting = $state(false);
  let autograding = $state(false);
  let actionOutput = $state('');
  let error = $state('');
  let successMsg = $state('');

  // Confirmation modal state
  let confirmState = $state({
    show: false,
    title: '',
    message: '',
    danger: false,
    onConfirm: () => {}
  });

  onMount(() => {
    if (!browser) return;
    if (!$authStore || $authStore.role !== 'admin') {
      window.location.href = '/';
    }
  });

  async function selectPool(pool: any) {
    selectedPool = { name: pool.name, userId: pool.userId };
    assignments = [];
    selectedAssignment = '';
    grades = [];
    actionOutput = '';
    error = '';
    await Promise.all([loadAssignments(), loadJupyterURL(), loadMoodleForPool()]);
  }

  async function loadJupyterURL() {
    if (!selectedPool) return;
    try {
      const res = await fetch(
        `/api/nbgrader/jupyter-url?pool_id=${encodeURIComponent(selectedPool.name)}&user_id=${encodeURIComponent(selectedPool.userId)}`
      );
      if (res.ok) {
        const data = await res.json();
        jupyterURL = data.url ?? '';
        jupyterDirectURL = (data.directUrl ?? '') + '/lab';
        // Direct VM URL for new-tab links (no proxy base_url configured).
        formgraderBaseURL = (data.directUrl ?? '').replace(/\/$/, '').replace(/%40/g, '@');
      }
    } catch { jupyterURL = ''; }
  }

  function openFormgrader() {
    if (formgraderBaseURL) window.open(`${formgraderBaseURL}/formgrader`, '_blank', 'noopener');
  }

  // Aggregate stats for the dashboard (right panel).
  // "missing" = pas de soumission : exclu des stats (sinon faux 0 qui fausse la moyenne).
  let submittedGrades = $derived(grades.filter(g => g.status !== 'missing'));
  let gradedCount = $derived(submittedGrades.length);
  let missingCount = $derived(grades.filter(g => g.status === 'missing').length);
  let manualCount = $derived(grades.filter(g => g.status === 'needs_manual_grade').length);
  let avgScore = $derived(submittedGrades.length ? submittedGrades.reduce((a, g) => a + g.score, 0) / submittedGrades.length : 0);
  // Copie triée — NE PAS faire grades.sort() dans le {#each} (mute l'état pendant le rendu
  // → erreur Svelte 5 state_unsafe_mutation qui gèle l'interactivité de la page).
  let sortedGrades = $derived([...grades].sort((a, b) => b.score - a.score));
  // Score-percentage distribution in 5 buckets (0-20 … 80-100).
  let distribution = $derived.by(() => {
    const buckets = [0, 0, 0, 0, 0];
    for (const g of submittedGrades) {
      if (g.max_score <= 0) continue;
      const pct = Math.max(0, Math.min(1, g.score / g.max_score));
      buckets[Math.min(4, Math.floor(pct * 5))]++;
    }
    return buckets;
  });

  async function loadAssignments() {
    if (!selectedPool) return;
    loadingAssignments = true;
    try {
      const res = await fetch(
        `/api/nbgrader/assignments?pool_id=${encodeURIComponent(selectedPool.name)}&user_id=${encodeURIComponent(selectedPool.userId)}`
      );
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      assignments = data.assignments ?? [];
    } catch (e: any) {
      error = e.message;
    } finally {
      loadingAssignments = false;
    }
  }

  async function loadGrades() {
    if (!selectedPool || !selectedAssignment) return;
    loadingGrades = true;
    error = '';
    try {
      const res = await fetch(
        `/api/nbgrader/grades?pool_id=${encodeURIComponent(selectedPool.name)}&user_id=${encodeURIComponent(selectedPool.userId)}&assignment=${encodeURIComponent(selectedAssignment)}`
      );
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      grades = data.grades ?? [];
    } catch (e: any) {
      error = e.message;
    } finally {
      loadingGrades = false;
    }
  }

  // Open the manual-grading page for a student's submission. Formgrader grades
  // a submission at /formgrader/submissions/<uuid>/?index=0 — we resolve that
  // uuid from the gradebook; if unavailable we fall back to the assignment's
  // submissions list (there is no /manage_submissions/<assignment>/<student> route).
  async function openManualGrading(student: string) {
    if (!selectedPool || !selectedAssignment) return;
    let url = `${formgraderBaseURL}/formgrader/manage_submissions/${encodeURIComponent(selectedAssignment)}`;
    try {
      const res = await fetch(
        `/api/nbgrader/submission-url?pool_id=${encodeURIComponent(selectedPool.name)}&user_id=${encodeURIComponent(selectedPool.userId)}&assignment=${encodeURIComponent(selectedAssignment)}&student=${encodeURIComponent(student)}`
      );
      if (res.ok) {
        const data = await res.json();
        if (data.submission_id) url = `${formgraderBaseURL}/formgrader/submissions/${data.submission_id}/?index=0`;
      }
    } catch { /* fall back to the submissions list */ }
    window.open(url, '_blank', 'noopener');
  }

  async function executeAction(endpoint: string, setter: (v: boolean) => void) {
    setter(true);
    actionOutput = '';
    error = '';
    successMsg = '';
    try {
      const params = new URLSearchParams({
        pool_id: selectedPool!.name,
        user_id: selectedPool!.userId,
      });
      if (selectedAssignment) params.set('assignment', selectedAssignment);
      const res = await fetch(`/api/nbgrader/${endpoint}?${params}`, { method: 'POST' });
      const data = await res.json();
      actionOutput = data.output ?? data.message ?? '';
      if (data.status === 'ok' || data.distributed !== undefined) {
        if (endpoint === 'collect' || endpoint === 'release') await loadAssignments();
        if (endpoint === 'autograde') {
          await loadGrades();
          // Envoi automatique vers Moodle si activé.
          if (autoPushMoodle && selectedMoodleAssign) await pushToMoodle();
        }
        // Confirmation message per action
        const a = selectedAssignment ? ` « ${selectedAssignment} »` : '';
        if (endpoint === 'release') {
          successMsg = `Devoir${a} distribué à ${data.distributed ?? 0} étudiant(s) ✓`;
        } else if (endpoint === 'collect') {
          const n = (data.output ?? '').match(/Collected (\d+)/)?.[1] ?? '0';
          successMsg = `${n} copie(s)${a} collectée(s) ✓`;
        } else if (endpoint === 'autograde') {
          successMsg = `Notation automatique${a} terminée ✓`;
        } else {
          successMsg = 'Opération terminée ✓';
        }
      } else {
        error = data.output ?? data.message ?? `${endpoint} failed`;
      }
    } catch (e: any) {
      error = e.message;
    } finally {
      setter(false);
    }
  }

  function postAction(endpoint: string, setter: (v: boolean) => void, confirmMsg?: string, danger: boolean = false) {
    if (!selectedPool) return;
    if (confirmMsg) {
      confirmState = {
        show: true,
        title: endpoint === 'release' ? 'Distribuer' : endpoint === 'collect' ? 'Collecter' : 'Notation',
        message: confirmMsg,
        danger,
        onConfirm: () => executeAction(endpoint, setter)
      };
    } else {
      executeAction(endpoint, setter);
    }
  }


  function downloadCSV() {
    if (!selectedPool) return;
    const params = new URLSearchParams({
      pool_id: selectedPool.name,
      user_id: selectedPool.userId,
    });
    if (selectedAssignment) params.set('assignment', selectedAssignment);
    window.open(`/api/nbgrader/export-csv?${params}`, '_blank');
  }

  // ── Moodle ──
  let moodleCourseId = $state(0);
  let moodleUrl = $state('');
  let moodleAssignments = $state<{ id: number; name: string; max_grade: number }[]>([]);
  let selectedMoodleAssign = $state<number | null>(null);
  let moodlePushing = $state(false);
  let moodlePushMsg = $state('');
  let moodleConfigured = $state(false);
  let autoPushMoodle = $state(false);
  let moodleCourses = $state<{ id: number; shortname: string; fullname: string }[]>([]);
  let linkCourseId = $state<number | null>(null);
  let linking = $state(false);

  async function loadMoodleForPool() {
    moodleCourseId = 0; moodleAssignments = []; selectedMoodleAssign = null; moodlePushMsg = '';
    if (!selectedPool) return;
    try {
      const st = await fetch('/api/moodle/status');
      if (st.ok) { const sd = await st.json(); moodleConfigured = !!sd.configured; moodleUrl = sd.url ?? ''; }
      const r = await fetch(`/api/moodle/assignments?pool_id=${encodeURIComponent(selectedPool.name)}&user_id=${encodeURIComponent(selectedPool.userId)}`);
      if (r.ok) {
        const d = await r.json();
        moodleCourseId = d.course_id ?? 0;
        moodleAssignments = d.assignments ?? [];
        if (moodleAssignments.length === 1) selectedMoodleAssign = moodleAssignments[0].id;
      }
      // Si pas encore lié, charger la liste des cours pour permettre la liaison.
      if (moodleConfigured && moodleCourseId === 0) {
        const cr = await fetch('/api/moodle/courses');
        if (cr.ok) moodleCourses = (await cr.json()).courses ?? [];
      }
    } catch { /* ignore */ }
  }

  async function linkPoolToMoodle() {
    if (!selectedPool || !linkCourseId) return;
    linking = true;
    try {
      const r = await fetch('/api/moodle/link-pool', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pool_id: selectedPool.name, user_id: selectedPool.userId, course_id: linkCourseId }),
      });
      if (r.ok) await loadMoodleForPool();
    } catch { /* ignore */ } finally { linking = false; }
  }

  async function pushToMoodle() {
    if (!selectedPool || !selectedAssignment || !selectedMoodleAssign) return;
    moodlePushing = true; moodlePushMsg = '';
    try {
      const r = await fetch('/api/moodle/push-grades', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          pool_id: selectedPool.name, user_id: selectedPool.userId,
          assignment: selectedAssignment, moodle_assign_id: selectedMoodleAssign,
        }),
      });
      const d = await r.json();
      if (!r.ok) { moodlePushMsg = 'Erreur : ' + (d.error ?? 'échec'); return; }
      moodlePushMsg = `${d.pushed} note(s) envoyée(s)`
        + (d.skipped ? `, ${d.skipped} ignorée(s)` : '')
        + (d.failures?.length ? `, ${d.failures.length} échec(s)` : '');
    } catch { moodlePushMsg = "Erreur lors de l'envoi vers Moodle."; }
    finally { moodlePushing = false; }
  }


  function scoreColor(grade: Grade): string {
    if (grade.max_score === 0) return 'text-neutral-500';
    const pct = grade.score / grade.max_score;
    if (pct >= 0.8) return 'text-green-600 dark:text-green-400';
    if (pct >= 0.5) return 'text-amber-600 dark:text-amber-400';
    return 'text-red-600 dark:text-red-400';
  }

  function avg(): string {
    if (!submittedGrades.length) return '—';
    return (submittedGrades.reduce((a, g) => a + g.score, 0) / submittedGrades.length).toFixed(1);
  }
</script>

<svelte:head><title>Notation — CloudPoolManager</title></svelte:head>

<div class="h-[calc(100vh-8rem)] flex flex-col gap-4 animate-fade-up">

  <ConfirmModal
    bind:show={confirmState.show}
    title={confirmState.title}
    message={confirmState.message}
    danger={confirmState.danger}
    onConfirm={confirmState.onConfirm}
  />

  <!-- Header + pool selector -->
  <div class="flex items-center gap-4 flex-wrap">
    <div>
      <h1 class="text-2xl font-bold text-primary-800 dark:text-primary-300" style="font-family:'Source Sans 3',sans-serif;">Notation</h1>
    </div>

    <select
      onchange={(e) => {
        const idx = parseInt((e.target as HTMLSelectElement).value);
        if (isNaN(idx)) { selectedPool = null; return; }
        const pool = allPools[idx];
        if (pool) selectPool(pool);
      }}
      class="field max-w-xs ml-auto"
    >
      <option value="">— Sélectionner un pool —</option>
      {#each allPools as pool, i}
        <option value="{i}">{pool.name} ({pool.userId})</option>
      {/each}
    </select>

    {#if selectedPool}
      <select
        bind:value={selectedAssignment}
        onchange={() => loadGrades()}
        class="field max-w-xs"
        disabled={loadingAssignments}
      >
        <option value="">— Assignment —</option>
        {#each assignments as a}
          <option value={a}>{a}</option>
        {/each}
      </select>
    {/if}
  </div>

  {#if error}
    <div class="card px-4 py-2.5 border-red-200 bg-red-50 dark:bg-red-900/20 dark:border-red-800 text-red-700 dark:text-red-300 text-sm">{error}</div>
  {/if}
  {#if successMsg}
    <div class="card px-4 py-2.5 border-green-200 bg-green-50 dark:bg-green-900/20 dark:border-green-800 text-green-700 dark:text-green-300 text-sm flex items-center justify-between gap-3 animate-fade-in">
      <span class="font-medium">{successMsg}</span>
      <button onclick={() => successMsg = ''} class="text-green-600/70 hover:text-green-800 dark:hover:text-green-200 shrink-0" aria-label="Fermer">✕</button>
    </div>
  {/if}

  {#if selectedPool}
  <!-- Main layout: left panel + JupyterLab iframe -->
  <div class="flex gap-4 flex-1 min-h-0">

    <!-- Left panel: actions + grades -->
    <div class="w-80 shrink-0 flex flex-col gap-3 overflow-y-auto">

      <!-- Actions -->
      <div class="card p-4 space-y-3">
        <p class="section-label block mb-3">Actions</p>

        <button
          onclick={() => postAction('release', v => releasing = v, `Êtes-vous sûr de vouloir distribuer l'assignment "${selectedAssignment}" à tous les étudiants ?`)}
          disabled={releasing || !selectedAssignment}
          class="btn btn-secondary w-full text-sm justify-start gap-2"
          title="Copie le devoir chez tous les étudiants"
        >
          {#if releasing}
            <span class="w-3.5 h-3.5 border-2 border-neutral-400/40 border-t-neutral-600 rounded-full shrink-0" style="animation:spinnerGlow 0.6s linear infinite;"></span>
          {:else}
            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
            </svg>
          {/if}
          Distribuer aux étudiants
        </button>

        <button
          onclick={() => postAction('collect', v => collecting = v, `Êtes-vous sûr de vouloir collecter les copies pour "${selectedAssignment}" ? Les travaux seront copiés depuis l'environnement des étudiants.`)}
          disabled={collecting || !selectedAssignment}
          class="btn btn-secondary w-full text-sm justify-start gap-2"
          title="Collecte les soumissions des étudiants"
        >
          {#if collecting}
            <span class="w-3.5 h-3.5 border-2 border-neutral-400/40 border-t-neutral-600 rounded-full shrink-0" style="animation:spinnerGlow 0.6s linear infinite;"></span>
          {:else}
            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
            </svg>
          {/if}
          Collecter les copies
        </button>

        <button
          onclick={() => postAction('autograde', v => autograding = v, `Êtes-vous sûr de vouloir lancer la notation automatique pour "${selectedAssignment}" ? Cela peut prendre plusieurs minutes.`, false)}
          disabled={autograding || !selectedAssignment}
          class="btn btn-primary w-full text-sm justify-start gap-2"
          title="Note automatiquement les notebooks"
        >
          {#if autograding}
            <span class="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full shrink-0" style="animation:spinnerGlow 0.6s linear infinite;"></span>
          {:else}
            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"/>
            </svg>
          {/if}
          Notation automatique
        </button>

        <button
          onclick={downloadCSV}
          class="btn btn-secondary w-full text-sm justify-start gap-2"
        >
          <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
          </svg>
          Exporter CSV
        </button>

        {#if moodleCourseId > 0}
          <div class="mt-1 p-3 rounded border border-[#f98012]/30 bg-[#f98012]/5 space-y-2">
            <p class="section-label flex items-center gap-1.5">
              <svg class="w-3.5 h-3.5 text-[#f98012]" fill="currentColor" viewBox="0 0 24 24"><path d="M12 3 1 9l4 2.18v6L12 21l7-3.82v-6l2-1.09V17h2V9L12 3z"/></svg>
              Moodle
            </p>
            <select class="field text-xs" bind:value={selectedMoodleAssign}>
              <option value={null} disabled selected>— Devoir Moodle cible —</option>
              {#each moodleAssignments as a}
                <option value={a.id}>{a.name} (/{a.max_grade})</option>
              {/each}
            </select>
            <button
              onclick={pushToMoodle}
              disabled={moodlePushing || !selectedAssignment || !selectedMoodleAssign}
              class="btn btn-secondary w-full text-xs justify-center gap-2"
            >
              {#if moodlePushing}
                <span class="w-3.5 h-3.5 border-2 border-neutral-400/40 border-t-neutral-600 rounded-full" style="animation: spinnerGlow 0.7s linear infinite;"></span>
              {/if}
              Envoyer les notes vers Moodle
            </button>
            <label class="flex items-center gap-2 text-xs text-neutral-600 cursor-pointer">
              <input type="checkbox" bind:checked={autoPushMoodle} class="w-3.5 h-3.5 accent-[#f98012]" />
              Envoyer automatiquement après « Notation automatique »
            </label>
            {#if moodlePushMsg}
              <p class="text-xs {moodlePushMsg.startsWith('Erreur') ? 'text-red-600' : 'text-green-600'}">{moodlePushMsg}</p>
            {/if}
            {#if moodleUrl}
              <a href={`${moodleUrl}/grade/report/grader/index.php?id=${moodleCourseId}`} target="_blank" rel="noopener noreferrer"
                 class="text-xs text-[#f98012] hover:underline inline-flex items-center gap-1">
                Carnet de notes Moodle ↗
              </a>
            {/if}
          </div>
        {:else if moodleConfigured && selectedPool}
          <div class="mt-1 p-3 rounded border border-neutral-200 bg-neutral-50 space-y-2">
            <p class="section-label flex items-center gap-1.5">
              <svg class="w-3.5 h-3.5 text-[#f98012]" fill="currentColor" viewBox="0 0 24 24"><path d="M12 3 1 9l4 2.18v6L12 21l7-3.82v-6l2-1.09V17h2V9L12 3z"/></svg>
              Moodle
            </p>
            <p class="text-xs text-neutral-500">Ce pool n'est lié à aucun cours Moodle. Liez-le pour envoyer les notes.</p>
            <select class="field text-xs" bind:value={linkCourseId}>
              <option value={null} disabled selected>— Cours Moodle —</option>
              {#each moodleCourses as c}
                <option value={c.id}>{c.shortname} — {c.fullname}</option>
              {/each}
            </select>
            <button onclick={linkPoolToMoodle} disabled={linking || !linkCourseId} class="btn btn-secondary w-full text-xs justify-center gap-2">
              {#if linking}<span class="w-3.5 h-3.5 border-2 border-neutral-400/40 border-t-neutral-600 rounded-full" style="animation: spinnerGlow 0.7s linear infinite;"></span>{/if}
              Lier ce pool au cours
            </button>
          </div>
        {/if}

        {#if actionOutput}
          <details class="mt-1">
            <summary class="text-xs text-neutral-500 cursor-pointer">Voir la sortie</summary>
            <pre class="mt-1 text-xs bg-neutral-900 text-green-400 p-2 rounded overflow-x-auto whitespace-pre-wrap max-h-32">{actionOutput}</pre>
          </details>
        {/if}
      </div>

      <!-- Grades table -->
      <div class="card overflow-hidden flex-1 min-h-0 flex flex-col">
        <div class="px-4 py-3 bg-neutral-50 dark:bg-neutral-800 border-b border-neutral-200 dark:border-neutral-700 flex items-center justify-between shrink-0">
          <p class="text-xs font-bold text-neutral-700 dark:text-neutral-300">
            Notes{selectedAssignment ? ` — ${selectedAssignment}` : ''}
          </p>
          {#if grades.length > 0}
            <span class="text-xs text-neutral-500">moy. {avg()}</span>
          {/if}
          {#if loadingGrades}
            <div class="w-3.5 h-3.5 rounded-full border-2 border-neutral-200 border-t-primary-700" style="animation:spinnerGlow 0.7s linear infinite;"></div>
          {/if}
        </div>

        <div class="overflow-y-auto flex-1">
          {#if grades.length === 0}
            <div class="flex flex-col items-center justify-center py-10 text-neutral-400 text-center px-4">
              <svg class="w-8 h-8 mb-2 text-neutral-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2"/>
              </svg>
              <p class="text-xs">
                {#if !selectedAssignment}Choisissez un assignment
                {:else}Lancez "Notation automatique"{/if}
              </p>
            </div>
          {:else}
            {#each sortedGrades as grade, i}
              <div class="px-4 py-2.5 border-b border-neutral-100 dark:border-neutral-800 last:border-0 animate-slide-right" style="animation-delay:{i*0.02}s">
                <div class="flex items-center justify-between mb-1">
                  <span class="text-xs font-mono text-neutral-800 dark:text-neutral-200 truncate max-w-[60%]">{grade.student}</span>
                  {#if grade.status === 'missing'}
                    <span class="text-xs font-semibold text-neutral-400">Non rendu</span>
                  {:else}
                    <span class="text-xs font-bold tabular-nums {scoreColor(grade)}">{grade.score.toFixed(1)}/{grade.max_score.toFixed(1)}</span>
                  {/if}
                </div>
                <div class="h-1.5 bg-neutral-200 dark:bg-neutral-700 rounded-full overflow-hidden mb-1.5">
                  <div
                    class="h-full rounded-full {grade.status === 'missing' ? 'bg-neutral-300 dark:bg-neutral-600' : grade.max_score > 0 && grade.score/grade.max_score >= 0.8 ? 'bg-green-500' : grade.max_score > 0 && grade.score/grade.max_score >= 0.5 ? 'bg-amber-500' : 'bg-red-500'}"
                    style="width:{grade.status === 'missing' ? 0 : (grade.max_score > 0 ? Math.round(grade.score/grade.max_score*100) : 0)}%"
                  ></div>
                </div>
                <div class="flex items-center justify-between">
                  {#if grade.status === 'needs_manual_grade'}
                    <span class="text-[10px] text-amber-600 dark:text-amber-400">Révision manuelle requise</span>
                  {:else if grade.status === 'missing'}
                    <span class="text-[10px] text-neutral-400">En attente de soumission</span>
                  {:else}
                    <span></span>
                  {/if}
                  <button
                    onclick={() => openManualGrading(grade.student)}
                    class="btn btn-secondary px-2.5 py-1 text-[10px] gap-1"
                  >
                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
                    Correction manuelle
                  </button>
                </div>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    </div>

    <!-- Espace de travail : lancement + tableau de bord -->
    <div class="flex-1 card overflow-hidden flex flex-col min-w-0">
      <div class="flex items-center justify-between px-4 py-2.5 bg-neutral-50 dark:bg-neutral-800 border-b border-neutral-200 dark:border-neutral-700 shrink-0">
        <span class="text-xs font-semibold text-neutral-700 dark:text-neutral-300">Espace de travail</span>
        {#if selectedPool}
          <span class="text-xs text-neutral-400 font-mono truncate max-w-48">{selectedPool.name}</span>
        {/if}
      </div>

      {#if jupyterDirectURL}
        <div class="flex-1 overflow-y-auto p-6 flex flex-col gap-6">
          <!-- Lancement (nouvel onglet) -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <a href={jupyterDirectURL} target="_blank" rel="noopener noreferrer" class="btn btn-primary justify-center gap-2 py-3">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>
              Ouvrir JupyterLab
            </a>
            <button onclick={openFormgrader} class="btn btn-secondary justify-center gap-2 py-3">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"/></svg>
              Ouvrir Formgrader
            </button>
          </div>
          <p class="text-xs text-neutral-400 -mt-3">JupyterLab et Formgrader s'ouvrent dans un onglet dédié.</p>

          {#if selectedAssignment && grades.length > 0}
            <!-- Tableau de bord notes -->
            <div>
              <p class="section-label mb-3">Vue d'ensemble — {selectedAssignment}</p>
              <div class="grid grid-cols-3 gap-3">
                <div class="card p-3 text-center"><p class="text-2xl font-bold text-primary-700 dark:text-primary-300 tabular-nums">{gradedCount}</p><p class="text-xs text-neutral-500">copies notées</p></div>
                <div class="card p-3 text-center"><p class="text-2xl font-bold text-primary-700 dark:text-primary-300 tabular-nums">{avgScore.toFixed(1)}</p><p class="text-xs text-neutral-500">moyenne</p></div>
                <div class="card p-3 text-center"><p class="text-2xl font-bold tabular-nums {manualCount > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-green-600 dark:text-green-400'}">{manualCount}</p><p class="text-xs text-neutral-500">à corriger</p></div>
              </div>
            </div>
            <!-- Distribution des notes -->
            <div>
              <p class="section-label mb-3">Distribution des notes</p>
              <div class="flex items-end gap-2 h-40">
                {#each distribution as count, i}
                  {@const maxC = Math.max(1, ...distribution)}
                  <div class="flex-1 flex flex-col items-center justify-end gap-1 h-full">
                    <span class="text-xs text-neutral-500 tabular-nums">{count}</span>
                    <div class="w-full rounded-t bg-primary-500/80 dark:bg-primary-400/80 transition-all" style="height:{Math.max(3, (count / maxC) * 100)}%"></div>
                    <span class="text-[10px] text-neutral-400 tabular-nums">{i * 20}–{i * 20 + 20}</span>
                  </div>
                {/each}
              </div>
              <p class="text-[10px] text-neutral-400 text-center mt-1">% du barème</p>
            </div>
          {:else}
            <!-- Guide du déroulé -->
            <div class="flex-1">
              <p class="section-label mb-3">Déroulé d'un devoir</p>
              <ol class="space-y-2.5 text-sm text-neutral-600 dark:text-neutral-300">
                <li class="flex gap-2.5"><span class="shrink-0 w-5 h-5 rounded-full bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-300 text-xs font-bold flex items-center justify-center">1</span><span>Ouvrir <b>JupyterLab</b> → menu <b>Nbgrader → Create Assignment</b>, marquer les cellules, puis <b>Generate</b>.</span></li>
                <li class="flex gap-2.5"><span class="shrink-0 w-5 h-5 rounded-full bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-300 text-xs font-bold flex items-center justify-center">2</span><span>Choisir l'assignment ci-dessus, puis <b>Distribuer aux étudiants</b>.</span></li>
                <li class="flex gap-2.5"><span class="shrink-0 w-5 h-5 rounded-full bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-300 text-xs font-bold flex items-center justify-center">3</span><span>Les étudiants éditent <code class="text-xs">nbgrader/&lt;devoir&gt;</code> puis cliquent <b>Soumettre</b>.</span></li>
                <li class="flex gap-2.5"><span class="shrink-0 w-5 h-5 rounded-full bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-300 text-xs font-bold flex items-center justify-center">4</span><span><b>Collecter les copies</b>, puis <b>Notation automatique</b>.</span></li>
                <li class="flex gap-2.5"><span class="shrink-0 w-5 h-5 rounded-full bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-300 text-xs font-bold flex items-center justify-center">5</span><span><b>Correction manuelle</b> (Formgrader) si besoin, puis <b>Exporter CSV</b>.</span></li>
              </ol>
            </div>
          {/if}
        </div>
      {:else}
        <div class="flex-1 flex flex-col items-center justify-center text-neutral-400 text-center gap-3 p-8">
          <svg class="w-14 h-14 text-neutral-200 dark:text-neutral-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
          </svg>
          <div>
            <p class="text-sm font-medium text-neutral-600 dark:text-neutral-400">JupyterLab non disponible</p>
            <p class="text-xs text-neutral-400 mt-1 max-w-xs">
              La VM enseignant doit être démarrée avec AppPort=8888.
            </p>
          </div>
        </div>
      {/if}
    </div>

  </div>
  {:else}
    <!-- No pool selected -->
    <div class="flex-1 card flex flex-col items-center justify-center text-center gap-4">
      <svg class="w-16 h-16 text-neutral-200 dark:text-neutral-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
      </svg>
      <div>
        <p class="text-base font-semibold text-neutral-600 dark:text-neutral-400">Sélectionnez un pool pour commencer</p>
        <p class="text-sm text-neutral-400 mt-1 max-w-sm mx-auto">
          Choisissez un pool dans le menu ci-dessus. La VM enseignant doit être démarrée avec AppPort=8888.
        </p>
      </div>
    </div>
  {/if}

</div>
