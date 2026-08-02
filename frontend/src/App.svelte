<script>
  import { onMount } from 'svelte'
  import { AddAccount, ListAccounts, LoginAccount, RemoveAccount, SetActiveAccount } from '../wailsjs/go/main/App.js'

  let accounts = []
  let name = ''
  let error = ''

  async function refresh() {
    try {
      accounts = await ListAccounts()
      error = ''
    } catch (err) {
      error = String(err)
    }
  }

  async function addAccount() {
    if (!name.trim()) return
    try {
      await AddAccount(name.trim())
      name = ''
      await refresh()
    } catch (err) {
      error = String(err)
    }
  }

  async function login(account) {
    try {
      await LoginAccount(account.id)
      await refresh()
    } catch (err) {
      error = String(err)
    }
  }

  async function activate(account) {
    try {
      await SetActiveAccount(account.id)
      await refresh()
    } catch (err) {
      error = String(err)
    }
  }

  async function remove(account) {
    try {
      await RemoveAccount(account.id)
      await refresh()
    } catch (err) {
      error = String(err)
    }
  }

  onMount(() => {
    refresh()
    const timer = setInterval(() => {
      if (accounts.some((account) => account.status === 'logging_in')) refresh()
    }, 1000)
    return () => clearInterval(timer)
  })
</script>

<main>
  <h1>账号管理</h1>
  <p class="hint">认证信息由官方登录流程管理，Mux 只保存账号元数据和独立 Profile。</p>

  <section class="add-account">
    <input bind:value={name} placeholder="账号名称" on:keydown={(event) => event.key === 'Enter' && addAccount()} />
    <button on:click={addAccount}>添加账号</button>
  </section>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  <section class="accounts">
    {#if accounts.length === 0}
      <p class="empty">还没有账号</p>
    {:else}
      {#each accounts as account}
        <article class:active={account.active}>
          <div>
            <strong>{account.name}</strong>
            <span>{account.status}</span>
            {#if account.error}<small class="account-error">{account.error}</small>{/if}
          </div>
          <div class="actions">
            {#if !account.active}<button on:click={() => activate(account)}>设为当前</button>{/if}
            {#if account.status !== 'logging_in'}
              <button on:click={() => login(account)}>登录</button>
            {:else}
              <span class="logging">登录中…</span>
            {/if}
            <button class="danger" on:click={() => remove(account)}>删除</button>
          </div>
        </article>
      {/each}
    {/if}
  </section>
</main>

<style>
  main { max-width: 760px; margin: 0 auto; padding: 48px; color: #222; }
  h1 { margin-bottom: 8px; }
  .hint { color: #666; }
  .add-account, .actions { display: flex; gap: 8px; }
  .add-account { margin: 32px 0 20px; }
  input { flex: 1; padding: 10px; border: 1px solid #ccc; border-radius: 6px; }
  button { padding: 9px 14px; border: 0; border-radius: 6px; cursor: pointer; }
  article { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 16px; margin: 10px 0; border: 1px solid #ddd; border-radius: 8px; }
  article.active { border-color: #4f7cff; }
  article span { display: block; margin-top: 5px; color: #777; font-size: 13px; }
  .danger { color: #b42318; }
  .error { color: #b42318; }
  .account-error { display: block; max-width: 520px; color: #b42318; font-size: 12px; }
  .empty { color: #777; }
</style>
