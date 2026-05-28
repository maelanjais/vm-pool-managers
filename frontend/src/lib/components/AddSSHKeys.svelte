<script lang="ts">
  import {
    ListStudentsRequestSchema, type ListStudentsRequest, type ListStudentsResponse,
    AddStudentRequestSchema, type AddStudentRequest, type AddStudentResponse,
    DeleteStudentRequestSchema, type DeleteStudentRequest, type DeleteStudentResponse,
  } from '$lib/grpc/frontcontrol_pb';
  import { addStudents, listStudents, deleteStudent } from '$lib/index';
  import { create } from '@bufbuild/protobuf';
  import { authStore } from '$lib/store';

  let {
    open = $bindable(),
    poolname,
  }: { open: boolean; poolname: string } = $props();

  let addModal = $state(false);
  let loading = $state(false);
  let error: string | null = $state(null);
  let rawMode = $state(false);
  let rawInput = $state('');

  interface User { name: string; sshKey: string; ip: string; }
  interface NewStudent { firstName: string; lastName: string; sshKey: string; }

  let users: User[] = $state([]);
  let newStudents: NewStudent[] = $state([{ firstName: '', lastName: '', sshKey: '' }]);

  function addRow() { newStudents = [...newStudents, { firstName: '', lastName: '', sshKey: '' }]; }
  function removeRow(i: number) { newStudents = newStudents.filter((_, idx) => idx !== i); }
  function buildLogin(s: NewStudent): string { return `${s.firstName.trim()}.${s.lastName.trim()}`.toLowerCase(); }

  async function handleListStudents() {
    const req: ListStudentsRequest = create(ListStudentsRequestSchema, { user: $authStore?.email, poolname });
    try {
      loading = true; error = null;
      const res: ListStudentsResponse = await listStudents(req);
      users = res.students.map(s => ({ name: s.name, sshKey: s.sshKey, ip: s.ip }));
    } catch { error = 'Erreur lors du chargement des étudiants.'; }
    finally { loading = false; }
  }

  async function handleDeleteStudent(name: string) {
    if (!confirm(`Supprimer l'étudiant ${name} ?`)) return;
    const req: DeleteStudentRequest = create(DeleteStudentRequestSchema, {
      user: $authStore?.email, poolname, studentName: name,
    });
    try {
      loading = true; error = null;
      await deleteStudent(req);
      await handleListStudents();
    } catch { error = "Erreur lors de la suppression."; }
    finally { loading = false; }
  }

  async function handleAdd() {
    const valid = newStudents.filter(s => s.firstName.trim() && s.lastName.trim() && s.sshKey.trim());
    if (!valid.length) { error = 'Aucun étudiant valide à ajouter.'; return; }
    const req: AddStudentRequest = create(AddStudentRequestSchema, {
      user: $authStore?.email, poolname,
      students: valid.map(s => ({ name: buildLogin(s), sshKey: s.sshKey })),
    });
    try {
      loading = true; error = null;
      await addStudents(req);
      await handleListStudents();
      newStudents = [{ firstName: '', lastName: '', sshKey: '' }];
      addModal = false;
    } catch { error = "Erreur lors de l'ajout."; }
    finally { loading = false; }
  }

  async function handleAddRaw() {
    const lines = rawInput.split('\n').map(l => l.trim()).filter(Boolean);
    const parsed: NewStudent[] = [];
    for (const line of lines) {
      const sep = line.indexOf(';');
      if (sep === -1) continue;
      const loginRaw = line.slice(0, sep).trim();
      const sshKey = line.slice(sep + 1).trim();
      if (!loginRaw || !sshKey) continue;
      const dotIdx = loginRaw.indexOf('.');
      const firstName = dotIdx !== -1 ? loginRaw.slice(0, dotIdx) : loginRaw;
      const lastName = dotIdx !== -1 ? loginRaw.slice(dotIdx + 1) : '';
      parsed.push({ firstName, lastName, sshKey });
    }
    if (!parsed.length) { error = 'Aucun étudiant valide (format: prenom.nom;cle_ssh)'; return; }
    newStudents = parsed;
    await handleAdd();
    rawInput = '';
    newStudents = [{ firstName: '', lastName: '', sshKey: '' }];
    addModal = false;
  }

  $effect(() => { if (open) handleListStudents(); });
  $effect(() => { if (rawMode) newStudents = [{ firstName: '', lastName: '', sshKey: '' }]; });
</script>

<!-- Main modal -->
{#if open}
  <div class="modal-overlay" role="dialog" aria-modal="true">
    <div class="modal-box" style="max-width:520px;">
      <div class="flex items-center justify-between mb-5">
        <div>
          <h3 class="text-base font-bold text-neutral-900" style="font-family: 'Source Sans 3', sans-serif;">Étudiants</h3>
          <p class="text-xs text-neutral-500 mt-0.5">{poolname}</p>
        </div>
        <button onclick={() => open = false} class="text-neutral-400 hover:text-neutral-700 transition-colors p-1 rounded hover:bg-neutral-100">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      {#if error}
        <div class="mb-4 px-3 py-2.5 rounded bg-red-50 border border-red-200 text-red-700 text-sm animate-fade-in">{error}</div>
      {/if}

      {#if loading}
        <div class="flex items-center justify-center py-16">
          <div class="w-8 h-8 rounded-full border-2 border-neutral-200 border-t-primary-700" style="animation: spinnerGlow 0.7s linear infinite;"></div>
        </div>
      {:else if users.length === 0}
        <div class="flex flex-col items-center justify-center py-14 text-neutral-400">
          <svg class="w-10 h-10 mb-3 text-neutral-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
          <p class="text-sm">Aucun étudiant enregistré</p>
        </div>
      {:else}
        <div class="space-y-1 max-h-72 overflow-y-auto pr-1 mb-4">
          {#each users as user, i}
            <div
              class="flex items-center justify-between px-4 py-3 rounded border border-neutral-100 bg-neutral-50 animate-slide-right"
              style="animation-delay:{i*0.03}s"
            >
              <div>
                <p class="text-sm font-semibold text-neutral-900">{user.name}</p>
                {#if user.ip}
                  <p class="text-xs text-neutral-500 font-mono mt-0.5">{user.ip}</p>
                {/if}
              </div>
              <div class="flex items-center gap-2">
                {#if user.ip}
                  <span class="badge badge-ready">Attribué</span>
                {:else}
                  <span class="badge badge-starting">En attente</span>
                {/if}
                <button
                  onclick={() => handleDeleteStudent(user.name)}
                  class="p-1.5 rounded text-neutral-400 hover:text-red-600 hover:bg-red-50 transition-colors"
                  title="Supprimer"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                  </svg>
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}

      <button onclick={() => addModal = true} class="btn btn-primary text-sm w-full">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
        </svg>
        Ajouter des étudiants
      </button>
    </div>
  </div>
{/if}

<!-- Add students modal -->
{#if addModal}
  <div class="modal-overlay" style="z-index:60;" role="dialog" aria-modal="true">
    <div class="modal-box" style="max-width:600px;">
      <div class="flex items-center justify-between mb-5">
        <h3 class="text-base font-bold text-neutral-900" style="font-family: 'Source Sans 3', sans-serif;">Ajouter des étudiants</h3>
        <button onclick={() => addModal = false} class="text-neutral-400 hover:text-neutral-700 transition-colors p-1 rounded hover:bg-neutral-100">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <!-- Mode toggle -->
      <div class="flex gap-1 mb-5 p-1 bg-neutral-100 rounded border border-neutral-200 w-fit">
        <button
          onclick={() => rawMode = false}
          class="px-4 py-1.5 rounded text-sm font-semibold transition-all {!rawMode ? 'bg-white text-primary-700 shadow-sm border border-neutral-200' : 'text-neutral-500 hover:text-neutral-700'}"
        >Formulaire</button>
        <button
          onclick={() => rawMode = true}
          class="px-4 py-1.5 rounded text-sm font-semibold transition-all {rawMode ? 'bg-white text-primary-700 shadow-sm border border-neutral-200' : 'text-neutral-500 hover:text-neutral-700'}"
        >Import texte</button>
      </div>

      {#if rawMode}
        <div class="space-y-3">
          <label class="section-label block mb-1">Un étudiant par ligne : <code class="text-primary-700 font-mono">prenom.nom;cle_ssh</code></label>
          <textarea
            class="field font-mono text-xs resize-none"
            rows="10"
            placeholder={"jean.dupont;ssh-ed25519 AAAA...\npaul.martin;ssh-ed25519 BBBB..."}
            bind:value={rawInput}
          ></textarea>
          <button onclick={handleAddRaw} disabled={!rawInput.trim() || loading} class="btn btn-primary text-sm">
            Importer
          </button>
        </div>
      {:else}
        <div class="space-y-3 max-h-72 overflow-y-auto pr-1">
          {#each newStudents as student, i}
            <div class="p-3 rounded border border-neutral-200 bg-neutral-50 space-y-2">
              <div class="flex gap-2">
                <input class="field flex-1" type="text" placeholder="Prénom" bind:value={student.firstName} />
                <input class="field flex-1" type="text" placeholder="Nom" bind:value={student.lastName} />
              </div>
              <div class="flex gap-2">
                <input class="field flex-1 font-mono text-xs" type="text" placeholder="ssh-ed25519 AAAA..." bind:value={student.sshKey} />
                {#if newStudents.length > 1}
                  <button onclick={() => removeRow(i)} class="btn btn-danger p-2 shrink-0">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                    </svg>
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
        <div class="flex justify-between items-center mt-4">
          <button onclick={addRow} class="btn btn-secondary text-sm">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
            </svg>
            Ajouter une ligne
          </button>
          <button onclick={handleAdd} disabled={loading} class="btn btn-primary text-sm">
            {#if loading}
              <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full" style="animation: spinnerGlow 0.6s linear infinite;"></span>
            {/if}
            Enregistrer
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}
