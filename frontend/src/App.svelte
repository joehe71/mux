<script>
  import { onMount } from 'svelte'
  import {
    AddAccount,
    CancelLogin,
    ListAccounts,
    LoginAccount,
    RemoveAccount,
    UpdateAccount,
  } from '../wailsjs/go/main/App.js'

  /** @typedef {{ id: string, name: string, email?: string, avatarUrl?: string, planType?: string, profilePath: string, status: string, createdAt: string, lastUsedAt?: string, error?: string, active: boolean }} Account */

  /** @type {Account[]} */
  let accounts = []
  let error = ''
  let saving = false
  let openMenuId = ''
  let updatingAccountId = ''
  let toast = ''

  async function refresh() {
    try {
      accounts = await ListAccounts()
      error = ''
    } catch (err) {
      error = String(err)
    }
  }

  /** @param {MouseEvent} event */
  function closeMenuOnOutsideClick(event) {
    const target = /** @type {HTMLElement | null} */ (event.target)
    if (!target?.closest('.card-menu')) openMenuId = ''
  }

  async function addAccount() {
    if (saving) return
    saving = true
    error = ''
    try {
      const added = await AddAccount('')
      accounts = await ListAccounts()
      if (added) await login(added)
    } catch (err) {
      error = String(err)
    } finally {
      saving = false
    }
  }

  /** @param {Account} account */
  async function login(account) {
    openMenuId = ''
    error = ''
    try {
      await LoginAccount(account.id)
      await refresh()
    } catch (err) {
      error = String(err)
    }
  }

  /** @param {Account} account */
  async function updateAccount(account) {
    if (updatingAccountId) return
    openMenuId = ''
    error = ''
    toast = ''
    updatingAccountId = account.id
    try {
      await UpdateAccount(account.id)
      await refresh()
      toast = '账号信息更新完成'
      setTimeout(() => {
        toast = ''
      }, 2500)
    } catch (err) {
      error = String(err)
    } finally {
      updatingAccountId = ''
    }
  }

  async function syncProfiles() {
    const currentAccounts = await ListAccounts()
    await Promise.all(
      currentAccounts.map(async (account) => {
        try {
          await UpdateAccount(account.id)
        } catch {
          // 没有凭据或 token 已失效时，保留现有本地资料。
        }
      }),
    )
    await refresh()
  }

  /** @param {Account} account */
  async function cancelLogin(account) {
    try {
      await CancelLogin(account.id)
    } catch (err) {
      // 登录可能已在用户点击前结束，取消操作应保持幂等。
      if (!String(err).includes('not in progress')) error = String(err)
    } finally {
      await refresh()
    }
  }

  /** @param {Account} account */
  async function remove(account) {
    try {
      await RemoveAccount(account.id)
      openMenuId = ''
      await refresh()
    } catch (err) {
      error = String(err)
    }
  }

  /** @param {Account} account */
  function statusText(account) {
    if (account.status === 'logging_in') return '登录中…'
    if (account.error || account.status === 'error') return '登录失败'
    if (account.status === 'expired') return '已过期'
    if (account.status === 'creating') return '初始化中…'
    return '正常'
  }

  /** @param {string | undefined} value */
  function formatDate(value) {
    if (!value) return '暂无记录'
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? '暂无记录' : date.toLocaleDateString('zh-CN')
  }

  onMount(() => {
    refresh().then(() => syncProfiles())
    const timer = setInterval(() => {
      if (accounts.some((account) => account.status === 'logging_in')) refresh()
    }, 1000)
    return () => clearInterval(timer)
  })
</script>

<svelte:window on:click={closeMenuOnOutsideClick} />

<main class="min-h-screen bg-white px-5 py-6 sm:px-8 sm:py-8 lg:px-12 lg:py-10">
  <section class="min-h-screen w-full bg-white sm:min-h-0">
    <header class="flex flex-wrap items-center gap-3 sm:gap-4">
      <div
        class="grid size-12 shrink-0 place-items-center rounded-2xl bg-indigo-600 text-xl font-bold text-white shadow-lg shadow-indigo-200"
      >
        M
      </div>
      <div>
        <p class="mb-0.5 text-xs font-bold tracking-[0.2em] text-slate-400">MUX</p>
        <h1 class="text-xl font-bold tracking-tight text-slate-900 sm:text-2xl">Codex 账号</h1>
      </div>
      <span class="ml-auto rounded-full bg-slate-100 px-3 py-1.5 text-xs font-medium text-slate-500"
        >{accounts.length} 个账号</span
      >
    </header>

    <div class="my-8 h-px bg-slate-100"></div>

    {#if error}
      <p
        class="mb-5 rounded-xl border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-600"
        role="alert"
      >
        {error}
      </p>
    {/if}

    <section
      class="grid grid-cols-1 gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3 xl:grid-cols-4"
      aria-label="Codex 账号列表"
    >
      {#each accounts as account (account.id)}
        <article
          class="group relative flex min-h-[260px] flex-col rounded-2xl border bg-white p-4 transition hover:-translate-y-0.5 hover:shadow-lg hover:shadow-slate-200/70 sm:min-h-[280px] sm:p-5"
          class:border-indigo-300={account.active}
          class:shadow-lg={account.active}
          class:shadow-indigo-100={account.active}
          class:border-slate-200={!account.active}
        >
          <div class="flex items-start justify-between gap-4">
            <div class="flex min-w-0 items-center gap-4">
              {#if account.avatarUrl}
                <img
                  class="size-16 shrink-0 rounded-2xl object-cover"
                  src={account.avatarUrl}
                  alt={`${account.name} 头像`}
                />
              {:else}
                <span
                  class="grid size-16 shrink-0 place-items-center rounded-2xl bg-indigo-100 text-2xl font-semibold text-indigo-600"
                  >{account.name.slice(0, 1).toUpperCase()}</span
                >
              {/if}
              <div class="min-w-0">
                <h2 class="truncate text-lg font-bold text-slate-900">{account.name}</h2>
                <p class="mt-1 truncate text-sm text-slate-500">{account.email || '暂无邮箱'}</p>
                <p class="mt-1 text-xs text-slate-400">套餐：{account.planType || '未知'}</p>
              </div>
            </div>
            <div class="card-menu relative">
              <button
                class="grid size-8 place-items-center rounded-lg text-sm tracking-widest text-slate-400 transition hover:bg-slate-100 hover:text-indigo-600"
                aria-label={`打开 ${account.name} 操作菜单`}
                aria-expanded={openMenuId === account.id}
                on:click|stopPropagation={() =>
                  (openMenuId = openMenuId === account.id ? '' : account.id)}>•••</button
              >
              {#if openMenuId === account.id}
                <div
                  class="absolute right-0 top-10 z-10 w-40 rounded-xl border border-slate-200 bg-white p-1.5 shadow-xl shadow-slate-300/40"
                >
                  <button
                    class="block w-full rounded-lg px-3 py-2 text-left text-xs text-slate-600 hover:bg-slate-50"
                    on:click={() => updateAccount(account)}>更新账号信息</button
                  >
                  <button
                    class="block w-full rounded-lg px-3 py-2 text-left text-xs text-red-500 hover:bg-red-50"
                    on:click={() => remove(account)}>删除账号</button
                  >
                </div>
              {/if}
            </div>
          </div>

          <div class="mt-5 flex flex-1 flex-col">
            <p class="flex items-center gap-2 text-xs" aria-label="账号状态">
              <span class="text-slate-400">账号状态：</span>
              <span
                class="size-2 rounded-full bg-emerald-400"
                class:bg-red-400={account.error}
                class:bg-amber-400={account.status === 'logging_in'}
              ></span>
              <span
                class:text-red-500={account.error}
                class:text-amber-500={account.status === 'logging_in'}>{statusText(account)}</span
              >
            </p>
            <dl class="mt-5 space-y-2 border-t border-slate-100 pt-4 text-[11px]">
              <div class="flex justify-between gap-3">
                <dt class="text-slate-400">创建时间</dt>
                <dd class="truncate text-slate-600">{formatDate(account.createdAt)}</dd>
              </div>
              <div class="flex justify-between gap-3">
                <dt class="text-slate-400">最近使用</dt>
                <dd class="truncate text-slate-600">{formatDate(account.lastUsedAt)}</dd>
              </div>
            </dl>
            {#if account.error}<small class="mt-2 block text-[11px] text-red-500"
                >{account.error}</small
              >{/if}
          </div>
          <div class="mt-5 flex items-center gap-3 border-t border-slate-100 pt-4">
            {#if account.status !== 'logging_in'}
              <button
                class="text-xs font-semibold text-indigo-600 hover:text-indigo-800"
                on:click={() => login(account)}>重新登录</button
              >
            {:else}<button
                class="text-xs font-semibold text-amber-600 hover:text-amber-800"
                on:click={() => cancelLogin(account)}>取消登录</button
              >{/if}
          </div>
          {#if updatingAccountId === account.id}
            <div
              class="absolute inset-0 z-20 grid place-items-center rounded-2xl bg-white/75 backdrop-blur-[1px]"
            >
              <div
                class="flex items-center gap-3 rounded-xl bg-white px-4 py-3 text-sm font-medium text-indigo-600 shadow-lg shadow-slate-200"
              >
                <span
                  class="size-5 animate-spin rounded-full border-2 border-indigo-200 border-t-indigo-600"
                ></span>
                更新中…
              </div>
            </div>
          {/if}
        </article>
      {/each}

      <button
        class="group flex min-h-[180px] flex-col items-center justify-center gap-2 rounded-2xl border border-dashed border-slate-300 bg-slate-50/60 text-slate-400 transition hover:-translate-y-0.5 hover:border-indigo-400 hover:bg-indigo-50/40 sm:min-h-[280px]"
        aria-label="新增 Codex 账号"
        on:click={addAccount}
      >
        <span
          class="text-5xl font-light leading-none text-indigo-500 transition group-hover:scale-110"
          >+</span
        >
        <strong class="text-sm text-slate-600">添加账号</strong>
        <span class="text-xs text-slate-400">登录新的 Codex 账号</span>
      </button>
    </section>

    {#if accounts.length === 0}<p class="mt-6 text-center text-sm text-slate-400">
        还没有账号，点击上方卡片开始登录
      </p>{/if}
  </section>
</main>

{#if toast}
  <div
    class="fixed right-5 top-5 z-50 rounded-xl border border-emerald-100 bg-white px-4 py-3 text-sm font-medium text-emerald-600 shadow-xl shadow-slate-200"
    role="status"
  >
    ✓ {toast}
  </div>
{/if}
