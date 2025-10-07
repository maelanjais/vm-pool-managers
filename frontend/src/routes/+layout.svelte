<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import logo from '$lib/assets/IDCS.png'
	import { authStore, tryLogin, logout , serverpoolStore } from '$lib/index'
	import { Navbar, NavBrand, NavLi, NavUl, NavHamburger, Button} from 'flowbite-svelte';
	import { Modal, Label, Input, Checkbox } from 'flowbite-svelte'

	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	
	let { children } = $props();

	onMount(() => {
		const token = get(authStore);
		if (token) {
			serverpoolStore.fetchInitData();
		} else {
			serverpoolStore.reset();
		}
	});

	// script pour modal login
	let loginModal = $state(false);
	let loginError = $state("");
	let loginSuccess = $state(false);


	async function handleLogin(event: Event) {
  		event.preventDefault();
 		const form = event.target as HTMLFormElement;
 		const data = new FormData(form);
		const email = data.get('email') as string;
		const password = data.get('password') as string;

		loginError = "";
		const result = await tryLogin(email, password);

		if (!result.success) {
			loginError = "Erreur lors du login";
			return;
  		}

		loginSuccess = true;
		setTimeout(() => {
    		form.reset();
			loginModal = false;
			loginSuccess = false;
		}, 3000);
	}

	// script pour modal createAccount
	let createAccountModal = $state(false);
	let createAccountError = $state("");
	let createAccountSuccess = $state(false);

	async function tryCreate(event:Event) {
		event.preventDefault();
		const form = event.target as HTMLFormElement;
		const data = new FormData(form);

		createAccountError = "";
		createAccountSuccess = false;
		if (!(data.get("name") as string) || !(data.get("email") as string) || !(data.get("password") as string) || !(data.get("confirmpassword") as string)) {
			createAccountError = "Champs non rempli";
			return false;
		}

		const password = (data.get("password") as string);
		const confirmpassword = (data.get("confirmpassword") as string);
		if (password !== confirmpassword) {
			createAccountError = "Les mots de passe ne correspondent pas";
			return false;
		}

		const jsonData = Object.fromEntries(data.entries());

		try {
			const response = await fetch('http://localhost:8080/users', {
				method:	'POST',
				headers:	{'Content-Type': 'application/json'},
				body: JSON.stringify(jsonData)
			});

			if (!response.ok) {
				createAccountError = "Erreur lors de la creation du compte";
				return
			}

			const result = await response.json();
			createAccountSuccess = true;
			
		} catch (err) {
			createAccountError = "Erreur backend";
			console.log(err);
		}

		setTimeout(() => {
			form.reset();
			createAccountModal = false;
			createAccountSuccess = false;
			loginModal = true;
		}, 3000);
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>


<!-- NavBar -->
<div class="min-h-screen bg-primary-500">
	<Navbar class=" sticky start-0 top-0 z-20 w-ful bg-tertiary-500 backdrop-blur-md shadow-md rounded-b-2xl">
		<NavBrand href="/">
			<img src={logo} class="me-3 h-6 sm:h-9" alt="ICDS Logo" />
			<span class="self-center text-xl font-semibold whitespace-nowrap text-gray-300 dark:text-white">CloudPoolManager</span>
		</NavBrand>
	<div class="flex md:order-2 gap-2">
		{#if $authStore}
			<Button size="sm" color="red" onclick={logout}>Deconnexion</Button>
		{:else}
		<Button size="sm" class="bg-secondary-500 border-white hover:bg-secondary-600" onclick={() => (loginModal = true)}>Login</Button>
		<Button size="sm" class="bg-option-500 border-white hover:bg-option-600" onclick={() => (createAccountModal = true)}>Create Account</Button>
		{/if}
		<NavHamburger />
	</div>
	<NavUl>
		<NavLi href="/" class="text-gray-300 text-xl">Home</NavLi>
		{#if $authStore}
		<NavLi href="/profile" class="text-gray-300 text-xl">Profil</NavLi>
		<NavLi href="/serverpools" class="text-gray-300 text-xl">Mes Serverpools</NavLi>
		{/if}
		<NavLi href="/" class="text-gray-300 text-xl">About</NavLi>
	</NavUl>
	
	</Navbar>
	<!-- Login Modal -->
	 <Modal bind:open={loginModal} class="bg-gray-400">
		<form class="flex flex-col space-y-6" onsubmit={handleLogin}>
			<h3 class="mb-4 text-2xl font-medium text-gray-800">Connexion</h3>
			{#if loginError}
				<Label color="red">{loginError}</Label>
			{/if}
			{#if loginSuccess}
				<Label color="green" class="text-xl">Connexion succès</Label>
			{/if}
			<Label class="space-y-2 text-xl">
				<span>Email</span>
				<Input type="email" name="email" placeholder="name@company.com" required/>
			</Label>
			<Label class="space-y-2 text-xl">
				<span>Password</span>
				<Input type="password" name="password" placeholder="votre mot de passe" required/>
			</Label>
			<Button type="submit">Se connecter</Button>
		</form>
	 </Modal>

	<!-- Create Account Modal -->
	<Modal bind:open={createAccountModal} class="bg-gray-400">
		<form class="flex flex-col space-y-6" onsubmit={tryCreate}>
			<h3 class="mb-4 text-2xl font-medium text-gray-800">Creer votre compte</h3>
			{#if createAccountError}
				<Label color="red">{createAccountError}</Label>
			{/if}
			{#if createAccountSuccess}
				<Label color="green" class="text-xl">Compte crée avec succès</Label>
			{/if}
			<Label class="space-y-2 text-xl">
				<span>Name</span>
				<Input type="text" name="name" placeholder="votre nom" required/>
			</Label>
			<Label class="space-y-2 text-xl">
				<span>Email</span>
				<Input type="email" name="email" placeholder="name@company.com" required/>
			</Label>
			<Label class="space-y-2 text-xl">
				<span>Password</span>
				<Input type="password" name="password" placeholder="votre mot de passe" required/>
			</Label>
			<Label class="space-y-2 text-xl">
				<span>Confirme Password</span>
				<Input type="password" name="confirmpassword" placeholder="Confirmez votre mot de passe" required/>
			</Label>
			<Button type="submit" class="bg-option-500">Creer</Button>
		</form>
	</Modal>

	<main class="pt-20 px-4 text-gray-300">
		{@render children?.()}
	</main>
</div>
