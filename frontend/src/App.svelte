<script>
  import { onMount } from 'svelte'
  import {
    AddAccount,
    CancelLogin,
    ListAccounts,
    LoginAccount,
    RemoveAccount,
    UpdateAccount,
    GetSyncInterval,
    SetSyncInterval,
    GetGatewayPort,
    SetGatewayPort,
    IsGatewayRunning,
    StartGateway,
    StopGateway,
    ConfigureFinchGateway,
    RemoveFinchGateway,
    OpenConfigFileFolder,
  } from '../wailsjs/go/main/App.js'

  /** @typedef {{ usedPercent: number, limitWindowSeconds: number, resetAt: number }} UsageWindow */
  /** @typedef {{ planType?: string, primaryWindow?: UsageWindow, secondaryWindow?: UsageWindow, limitReached?: boolean, hasCredits?: boolean, unlimited?: boolean, balance?: string }} Usage */
  /** @typedef {{ id: string, name: string, email?: string, avatarUrl?: string, planType?: string, usage?: Usage, usageUpdatedAt?: string, profilePath: string, status: string, createdAt: string, lastUsedAt?: string, error?: string, active: boolean }} Account */

  /** @type {Account[]} */
  let accounts = []
  let error = ''
  let saving = false
  let updatingAccountId = ''
  let toast = ''
  let showSettings = false
  /** @type {Account | null} */
  let selectedAccount = null
  let syncInterval = 10
  let gatewayPort = 8787
  let gatewayEnabled = false
  let showFinchMenu = false
  let showSettingsJson = false
  let gatewaySettingsExpanded = true
  let syncIntervalError = ''

  /** @param {MouseEvent} event */
  function closeFinchMenuOnOutsideClick(event) {
    const target = /** @type {HTMLElement | null} */ (event.target)
    if (!target?.closest('.finch-menu')) showFinchMenu = false
  }

  async function toggleGateway() {
    try {
      if (gatewayEnabled) {
        await StopGateway()
      } else {
        await StartGateway()
      }
      gatewayEnabled = !gatewayEnabled
    } catch (err) {
      error = String(err)
    }
  }

  $: weeklyUsage = accounts.reduce(
    (summary, account) => {
      const window = account.usage?.primaryWindow
      if (!window) return summary
      summary.total += 1
      summary.used += window.usedPercent
      summary.remaining += Math.max(0, 100 - window.usedPercent)
      if (window.usedPercent >= 100) summary.exhausted += 1
      return summary
    },
    { total: 0, used: 0, remaining: 0, exhausted: 0 },
  )

  async function refresh() {
    try {
      accounts = await ListAccounts()
      error = ''
    } catch (err) {
      error = String(err)
    }
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
      await refresh()
    } catch (err) {
      error = String(err)
    }
  }

  /** @param {number} value */
  function formatResetAt(value) {
    if (!value) return '暂无重置时间'
    return new Date(value * 1000).toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  /** @param {UsageWindow} window */
  function windowLabel(window) {
    const days = Math.round(window.limitWindowSeconds / 86400)
    return days >= 28 ? '月度窗口' : `${days || 7} 天窗口`
  }

  /** @param {string | undefined} value */
  function updatedLabel(value) {
    if (!value) return '尚未更新'
    return `更新于：${new Date(value).toLocaleString('zh-CN')}`
  }

  /** @param {Account} account */
  function statusText(account) {
    if (account.status === 'logging_in') return '登录中…'
    if (account.error || account.status === 'error') return '登录失败'
    if (account.status === 'expired') return '已过期'
    if (account.status === 'creating') return '初始化中…'
    return '正常'
  }

  onMount(() => {
    refresh().then(() => syncProfiles())
    const timer = setInterval(() => {
      if (accounts.some((account) => account.status === 'logging_in')) refresh()
    }, 1000)
    return () => clearInterval(timer)
  })
</script>

<svelte:window on:click={closeFinchMenuOnOutsideClick} />

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
      <div class="ml-auto flex items-center gap-2">
        <button
          class={`tooltip rounded-xl px-3 py-1.5 text-xs font-semibold transition ${gatewayEnabled ? 'bg-emerald-50 text-emerald-600 hover:bg-emerald-100' : 'border border-slate-200 text-slate-500 hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-600'}`}
          aria-label={gatewayEnabled ? '停止模型网关' : '启动模型网关'}
          class:tooltip={true}
          data-tooltip={gatewayEnabled ? '停止模型网关' : '启动模型网关'}
          on:click={toggleGateway}>{gatewayEnabled ? '网关运行中' : '启动网关'}</button
        >
        <div class="finch-menu relative flex">
          <button
            class="tooltip rounded-l-xl border border-r-0 border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-500 transition hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-600"
            aria-label="配置到 Finch"
            data-tooltip="一键配置到 Finch"
            on:click={async () => {
              try {
                await ConfigureFinchGateway()
                toast = '已配置到 Finch custom provider'
                setTimeout(() => (toast = ''), 2500)
              } catch (err) {
                error = String(err)
              }
            }}>接入 Finch</button
          >
          <button
            class="tooltip rounded-r-xl border border-slate-200 px-2 py-1.5 text-xs text-slate-500 transition hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-600"
            aria-label="打开 Finch 配置菜单"
            data-tooltip="更多 Finch 操作"
            on:click={() => (showFinchMenu = !showFinchMenu)}
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              class="size-4"
              aria-hidden="true"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="m6 9 6 6 6-6" />
            </svg>
          </button>
          {#if showFinchMenu}
            <div
              class="absolute right-0 top-full z-30 mt-2 w-36 rounded-xl border border-slate-200 bg-white p-1.5 shadow-xl shadow-slate-300/40"
            >
              <button
                class="block w-full rounded-lg px-3 py-2 text-left text-xs text-red-500 hover:bg-red-50"
                on:click={async () => {
                  try {
                    await RemoveFinchGateway()
                    showFinchMenu = false
                    toast = '已从 Finch 移除 Mux Gateway'
                    setTimeout(() => (toast = ''), 2500)
                  } catch (err) {
                    error = String(err)
                  }
                }}>移除 Finch 配置</button
              >
            </div>
          {/if}
        </div>
        <button
          class="tooltip rounded-xl border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-500 transition hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-600"
          aria-label="打开配置"
          data-tooltip="配置"
          on:click={async () => {
            syncInterval = await GetSyncInterval()
            gatewayPort = await GetGatewayPort()
            gatewayEnabled = await IsGatewayRunning()
            syncIntervalError = ''
            showSettings = true
          }}>⚙ 配置</button
        >
      </div>
    </header>

    <div class="my-8 h-px bg-slate-100"></div>

    <section
      class="mb-6 rounded-2xl border border-slate-200 bg-slate-50/70 px-5 py-4"
      aria-label="全部账号 7 天用量"
    >
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold text-slate-700">全部账号 7 天用量</h2>
          <p class="mt-1 text-xs text-slate-400">
            整体已用 {weeklyUsage.total ? (weeklyUsage.used / weeklyUsage.total).toFixed(0) : 0}% ·
            整体剩余 {weeklyUsage.total
              ? (weeklyUsage.remaining / weeklyUsage.total).toFixed(0)
              : 0}%
          </p>
        </div>
        <div class="w-full sm:w-56">
          <div class="mb-1 flex justify-between text-[11px] text-slate-400">
            <span>所有账号总配额</span>
            <span>{weeklyUsage.total ? (weeklyUsage.used / weeklyUsage.total).toFixed(0) : 0}%</span
            >
          </div>
          <div class="h-2 overflow-hidden rounded-full bg-slate-200">
            <div
              class="h-full rounded-full bg-indigo-500 transition-all"
              style={`width: ${weeklyUsage.total ? Math.min(100, weeklyUsage.used / weeklyUsage.total) : 0}%`}
            ></div>
          </div>
          <p
            class="mt-1 text-right text-xs font-semibold {weeklyUsage.exhausted > 0
              ? 'text-red-500'
              : 'text-slate-600'}"
          >
            已用完：{weeklyUsage.exhausted} 个账号
          </p>
        </div>
      </div>
    </section>

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
          class="group relative flex min-h-[260px] flex-col rounded-2xl border border-slate-200 bg-white p-4 transition hover:-translate-y-0.5 hover:shadow-lg hover:shadow-slate-200/70 sm:min-h-[280px] sm:p-5"
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
                <div class="group/email relative mt-1 min-w-0">
                  <p class="truncate text-sm text-slate-500">{account.email || '暂无邮箱'}</p>
                  {#if account.email}
                    <span
                      class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 hidden max-w-xs whitespace-nowrap rounded-lg bg-slate-800 px-3 py-1.5 text-xs text-white shadow-lg group-hover/email:block"
                      >{account.email}</span
                    >
                  {/if}
                </div>
                <p class="mt-1 text-xs text-slate-400">套餐：{account.planType || '未知'}</p>
              </div>
            </div>
            <div class="group/refresh relative">
              <button
                class="grid size-8 place-items-center rounded-lg text-xl leading-none text-slate-400 transition hover:bg-slate-100 hover:text-indigo-600 disabled:cursor-wait disabled:opacity-50"
                aria-label={`刷新 ${account.name} 用户信息`}
                disabled={Boolean(updatingAccountId)}
                on:click={() => updateAccount(account)}>↻</button
              >
              <span
                class="pointer-events-none absolute right-0 top-full z-30 mt-2 hidden whitespace-nowrap rounded-lg bg-slate-800 px-3 py-1.5 text-xs text-white shadow-lg group-hover/refresh:block"
                >{updatedLabel(account.usageUpdatedAt)}</span
              >
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
            <div
              class="mt-5 space-y-2 rounded-xl border border-slate-100 bg-slate-50/70 px-3 py-2.5"
            >
              {#if account.usage?.primaryWindow}
                <p class="text-xs text-slate-500">
                  {windowLabel(account.usage.primaryWindow)}：剩余 {Math.max(
                    0,
                    100 - account.usage.primaryWindow.usedPercent,
                  ).toFixed(0)}% · {formatResetAt(account.usage.primaryWindow.resetAt)} 重置
                </p>
              {/if}
              {#if account.usage?.secondaryWindow}
                <p class="text-xs text-slate-500">
                  {windowLabel(account.usage.secondaryWindow)}：剩余 {Math.max(
                    0,
                    100 - account.usage.secondaryWindow.usedPercent,
                  ).toFixed(0)}% · {formatResetAt(account.usage.secondaryWindow.resetAt)} 重置
                </p>
              {/if}
              {#if account.usage?.limitReached}<p class="text-xs font-medium text-red-500">
                  已达到当前限额
                </p>{/if}
              {#if account.usage?.unlimited}<p class="text-xs text-slate-500">Credits：无限</p>
              {:else if account.usage?.balance}<p class="text-xs text-slate-500">
                  Credits：{account.usage.balance}
                </p>{/if}
              {#if !account.usage?.primaryWindow && !account.usage?.secondaryWindow}<p
                  class="text-xs text-slate-400"
                >
                  暂无用量信息
                </p>{/if}
            </div>
            {#if account.error}<small class="mt-2 block text-[11px] text-red-500"
                >{account.error}</small
              >{/if}
          </div>
          <div class="mt-5 flex items-center justify-between border-t border-slate-100 pt-4">
            <div class="flex items-center gap-4">
              {#if account.status !== 'logging_in'}
                <button
                  class="text-xs font-semibold text-indigo-600 hover:text-indigo-800"
                  on:click={() => login(account)}>重新登录</button
                >
              {:else}<button
                  class="text-xs font-semibold text-amber-600 hover:text-amber-800"
                  on:click={() => cancelLogin(account)}>取消登录</button
                >{/if}
              <button
                class="text-xs font-semibold text-red-500 hover:text-red-700"
                on:click={() => remove(account)}>删除账号</button
              >
            </div>
            <button
              class="grid size-8 place-items-center rounded-lg text-xl leading-none text-slate-400 hover:bg-indigo-50 hover:text-indigo-600"
              aria-label={`查看 ${account.name} 详情`}
              class:tooltip={true}
              data-tooltip="查看账号详情"
              on:click={() => (selectedAccount = account)}>→</button
            >
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

  {#if selectedAccount}
    <div
      class="fixed inset-0 z-40 grid place-items-center bg-slate-900/20 px-5 backdrop-blur-[2px]"
      role="presentation"
      on:click={(event) => {
        if (event.target === event.currentTarget) selectedAccount = null
      }}
    >
      <div
        class="w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-6 shadow-2xl shadow-slate-300/50"
        role="dialog"
        aria-modal="true"
        aria-labelledby="account-details-title"
      >
        <div class="flex items-center justify-between">
          <h2 id="account-details-title" class="text-lg font-bold text-slate-900">账号详情</h2>
          <button
            class="grid size-8 place-items-center rounded-lg text-lg text-slate-400 hover:bg-slate-100 hover:text-slate-700"
            aria-label="关闭账号详情"
            on:click={() => (selectedAccount = null)}>×</button
          >
        </div>
        <section class="mt-5" aria-labelledby="account-info-title">
          <h3 id="account-info-title" class="mb-3 text-sm font-semibold text-slate-700">
            账号信息
          </h3>
          <div
            class="space-y-3 rounded-xl border border-slate-100 bg-slate-50/70 px-4 py-3 text-sm"
          >
            <div class="flex justify-between gap-4">
              <span class="text-slate-400">账号</span><span class="truncate text-slate-700"
                >{selectedAccount.name}</span
              >
            </div>
            <div class="flex justify-between gap-4">
              <span class="text-slate-400">邮箱</span><span class="truncate text-slate-700"
                >{selectedAccount.email || '暂无邮箱'}</span
              >
            </div>
            <div class="flex justify-between gap-4">
              <span class="text-slate-400">套餐</span><span class="text-slate-700"
                >{selectedAccount.planType || '未知'}</span
              >
            </div>
            <div class="flex justify-between gap-4">
              <span class="text-slate-400">状态</span><span class="text-slate-700"
                >{statusText(selectedAccount)}</span
              >
            </div>
            <div class="flex justify-between gap-4">
              <span class="text-slate-400">上次更新</span><span class="text-slate-700"
                >{updatedLabel(selectedAccount.usageUpdatedAt)}</span
              >
            </div>
          </div>
        </section>
        <section class="mt-6 border-t border-slate-100 pt-5" aria-labelledby="model-usage-title">
          <h3 id="model-usage-title" class="mb-3 text-sm font-semibold text-slate-700">
            模型用量信息
          </h3>
          <div class="space-y-3">
            {#if selectedAccount.usage?.primaryWindow}
              <div class="rounded-xl border border-slate-100 bg-slate-50/70 px-4 py-3">
                <div class="flex justify-between gap-4 text-sm">
                  <span class="text-slate-500"
                    >{windowLabel(selectedAccount.usage.primaryWindow)}</span
                  ><span class="font-semibold text-slate-700"
                    >已用 {selectedAccount.usage.primaryWindow.usedPercent.toFixed(0)}%</span
                  >
                </div>
                <p class="mt-1 text-xs text-slate-400">
                  {formatResetAt(selectedAccount.usage.primaryWindow.resetAt)} 重置
                </p>
              </div>
            {/if}
            {#if selectedAccount.usage?.secondaryWindow}
              <div class="rounded-xl border border-slate-100 bg-slate-50/70 px-4 py-3">
                <div class="flex justify-between gap-4 text-sm">
                  <span class="text-slate-500"
                    >{windowLabel(selectedAccount.usage.secondaryWindow)}</span
                  ><span class="font-semibold text-slate-700"
                    >已用 {selectedAccount.usage.secondaryWindow.usedPercent.toFixed(0)}%</span
                  >
                </div>
                <p class="mt-1 text-xs text-slate-400">
                  {formatResetAt(selectedAccount.usage.secondaryWindow.resetAt)} 重置
                </p>
              </div>
            {/if}
            {#if selectedAccount.usage?.limitReached}<p
                class="rounded-xl bg-red-50 px-4 py-3 text-xs font-medium text-red-500"
              >
                已达到当前限额
              </p>{/if}
            <div
              class="rounded-xl border border-slate-100 bg-slate-50/70 px-4 py-3 text-sm text-slate-500"
            >
              {#if selectedAccount.usage?.unlimited}Credits：无限{:else}Credits：{selectedAccount
                  .usage?.balance || '0'}{/if}
            </div>
            {#if !selectedAccount.usage?.primaryWindow && !selectedAccount.usage?.secondaryWindow}<p
                class="text-xs text-slate-400"
              >
                暂无模型用量信息
              </p>{/if}
          </div>
        </section>
      </div>
    </div>
  {/if}

  {#if showSettings}
    <div
      class="fixed inset-0 z-40 grid place-items-center bg-slate-900/20 px-5 backdrop-blur-[2px]"
      role="presentation"
      on:click={(event) => {
        if (event.target === event.currentTarget) showSettings = false
      }}
    >
      <div
        class="w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-6 shadow-2xl shadow-slate-300/50"
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-title"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <h2 id="settings-title" class="text-lg font-bold text-slate-900">配置</h2>
            <div class="group/config relative">
              <button
                class="grid size-8 place-items-center rounded-lg text-lg text-slate-400 hover:bg-indigo-50 hover:text-indigo-600"
                aria-label="打开应用目录"
                on:click={async () => {
                  try {
                    await OpenConfigFileFolder()
                  } catch (err) {
                    syncIntervalError = String(err)
                  }
                }}
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  class="size-5"
                  aria-hidden="true"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M3.75 6.75h5l1.5 2h10v8.5a1.5 1.5 0 0 1-1.5 1.5h-15a1.5 1.5 0 0 1-1.5-1.5v-9a1.5 1.5 0 0 1 1.5-1.5Z"
                  />
                </svg>
              </button>
              <span
                class="pointer-events-none absolute left-0 top-full z-30 mt-2 hidden whitespace-nowrap rounded-lg bg-slate-800 px-3 py-1.5 text-xs text-white shadow-lg group-hover/config:block"
              >
                打开应用目录
              </span>
            </div>
          </div>
          <button
            class="grid size-8 place-items-center rounded-lg text-lg text-slate-400 hover:bg-slate-100 hover:text-slate-700"
            aria-label="关闭配置面板"
            on:click={() => (showSettings = false)}>×</button
          >
        </div>
        <div class="mt-5 rounded-xl border border-slate-100 bg-slate-50 px-4 py-3">
          <p class="mb-3 border-b border-slate-200 pb-3 text-sm font-semibold text-slate-700">
            账号同步
          </p>
          <p class="mt-1 text-xs leading-5 text-slate-400">
            应用会按设定间隔自动更新所有账号信息和用量。
          </p>
          <label
            class="mt-4 flex items-center justify-between gap-4 text-sm text-slate-600"
            for="sync-interval"
          >
            <span>同步间隔</span>
            <span class="flex items-center gap-2">
              <input
                id="sync-interval"
                class="w-20 rounded-lg border border-slate-200 px-2 py-1.5 text-right text-sm outline-none focus:border-indigo-400"
                type="number"
                min="5"
                step="1"
                bind:value={syncInterval}
              />
              <span class="text-xs text-slate-400">分钟（最少 5 分钟）</span>
            </span>
          </label>
          <button
            class="mb-3 mt-6 flex w-full items-center justify-between border-b border-slate-200 pb-3 text-left text-sm font-semibold text-slate-700"
            aria-expanded={gatewaySettingsExpanded}
            on:click={() => (gatewaySettingsExpanded = !gatewaySettingsExpanded)}
          >
            网关配置
            <span class:rotate-180={gatewaySettingsExpanded}>⌄</span>
          </button>
          {#if gatewaySettingsExpanded}
            <label
              class="mt-4 flex items-center justify-between gap-4 text-sm text-slate-600"
              for="gateway-port"
            >
              <span>默认网关端口</span>
              <span class="flex items-center gap-2">
                <input
                  id="gateway-port"
                  class="w-20 rounded-lg border border-slate-200 px-2 py-1.5 text-right text-sm outline-none focus:border-indigo-400"
                  type="number"
                  min="1024"
                  max="65535"
                  step="1"
                  bind:value={gatewayPort}
                />
                <span class="text-xs text-slate-400">HTTP（1024-65535）</span>
              </span>
            </label>
          {/if}
          {#if syncIntervalError}<p class="mt-2 text-xs text-red-500">{syncIntervalError}</p>{/if}
        </div>
        {#if showSettingsJson}
          <pre
            class="mt-4 max-h-40 overflow-auto rounded-xl bg-slate-900 p-3 text-xs text-slate-200">{JSON.stringify(
              {
                syncIntervalMinutes: Number(syncInterval),
                gatewayPort: Number(gatewayPort),
                gatewayEnabled,
              },
              null,
              2,
            )}</pre>
        {/if}
        <div class="mt-6 flex justify-end gap-2">
          <button
            class="rounded-xl border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-50"
            on:click={() => (showSettingsJson = !showSettingsJson)}
            >{showSettingsJson ? '隐藏 JSON' : 'JSON'}</button
          >
          <button
            class="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
            on:click={async () => {
              if (syncInterval < 5) {
                syncIntervalError = '同步间隔不能低于 5 分钟'
                return
              }
              if (gatewayPort < 1024 || gatewayPort > 65535) {
                syncIntervalError = '网关端口必须在 1024-65535 之间'
                return
              }
              try {
                await SetSyncInterval(Number(syncInterval))
                await SetGatewayPort(Number(gatewayPort))
                showSettings = false
              } catch (err) {
                syncIntervalError = String(err)
              }
            }}>保存</button
          >
        </div>
      </div>
    </div>
  {/if}
</main>

{#if toast}
  <div
    class="fixed right-5 top-5 z-50 rounded-xl border border-emerald-100 bg-white px-4 py-3 text-sm font-medium text-emerald-600 shadow-xl shadow-slate-200"
    role="status"
  >
    ✓ {toast}
  </div>
{/if}
