import { useEffect, useMemo, useState } from "react"
import {
  Activity,
  CheckCircle2,
  Copy,
  Plus,
  RefreshCw,
  Search,
  ShieldAlert,
  Trash2,
  XCircle,
  Key,
  LogIn,
  Layers,
  X,
} from "lucide-react"
import { Button } from "../components/ui/button"
import { toast } from "sonner"
import { getAuthHeader } from "../lib/auth"
import { API_BASE } from "../lib/api"

type AccountItem = {
  email: string
  password?: string
  token?: string
  cookies?: string
  username?: string
  provider?: string
  valid?: boolean
  inflight?: number
  max_inflight?: number
  rate_limited_until?: number
  status_code?: string
  last_error?: string
  last_request_started?: number
  last_request_finished?: number
  consecutive_failures?: number
  rate_limit_strikes?: number
}

const PROVIDERS = [
  { id: "all", name: "Semua Provider", color: "bg-muted text-foreground" },
  { id: "deepseek", name: "DeepSeek AI", color: "bg-blue-500/10 text-blue-700 border-blue-500/30 dark:text-blue-400" },
  { id: "qwen", name: "Qwen AI", color: "bg-purple-500/10 text-purple-700 border-purple-500/30 dark:text-purple-400" },
  { id: "chatgpt", name: "ChatGPT (OpenAI)", color: "bg-emerald-500/10 text-emerald-700 border-emerald-500/30 dark:text-emerald-400" },
  { id: "mistral", name: "Mistral AI", color: "bg-amber-500/10 text-amber-700 border-amber-500/30 dark:text-amber-400" },
]

function getProviderInfo(provId?: string) {
  const p = PROVIDERS.find(item => item.id === provId)
  return p || { id: provId || "deepseek", name: provId || "DeepSeek", color: "bg-muted text-muted-foreground" }
}

function maskedToken(token?: string) {
  const value = (token || "").trim()
  if (!value) return "-"
  if (value.length <= 16) return "••••••••••••"
  return `${value.slice(0, 8)}••••••${value.slice(-6)}`
}

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<AccountItem[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedProviderTab, setSelectedProviderTab] = useState("all")
  const [searchQuery, setSearchQuery] = useState("")
  const [statusFilter, setStatusFilter] = useState("all")

  // Modal states
  const [modalOpen, setModalOpen] = useState(false)
  const [formProvider, setFormProvider] = useState("deepseek")
  const [loginMode, setLoginMode] = useState<"auto" | "token">("auto")
  const [formEmail, setFormEmail] = useState("")
  const [formPassword, setFormPassword] = useState("")
  const [formToken, setFormToken] = useState("")
  const [formCookies, setFormCookies] = useState("")
  const [formUsername, setFormUsername] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [verifyingEmail, setVerifyingEmail] = useState<string | null>(null)

  const fetchAccounts = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/accounts`, { headers: getAuthHeader() })
      if (!res.ok) throw new Error("Gagal mengambil akun")
      const data = await res.json()
      if (Array.isArray(data.accounts)) {
        setAccounts(data.accounts)
      }
    } catch {
      toast.error("Gagal memuat daftar akun dari server")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchAccounts()
  }, [])

  const openAddModal = (defaultProv = "deepseek") => {
    setFormProvider(defaultProv)
    setLoginMode(defaultProv === "deepseek" || defaultProv === "qwen" ? "auto" : "token")
    setFormEmail("")
    setFormPassword("")
    setFormToken("")
    setFormCookies("")
    setFormUsername("")
    setModalOpen(true)
  }

  const handleProviderSelectInModal = (provId: string) => {
    setFormProvider(provId)
    if (provId === "chatgpt" || provId === "mistral") {
      setLoginMode("token")
    } else {
      setLoginMode("auto")
    }
  }

  const handleSaveAccount = async () => {
    if (loginMode === "auto" && (!formEmail.trim() || !formPassword.trim())) {
      toast.error("Silakan masukkan Email dan Kata Sandi")
      return
    }
    if (loginMode === "token" && !formToken.trim()) {
      toast.error("Silakan masukkan Token atau Kredensial akun")
      return
    }

    setSubmitting(true)
    const toastId = toast.loading(
      loginMode === "auto"
        ? `Sedang memverifikasi & login ke ${getProviderInfo(formProvider).name}...`
        : `Menyimpan akun ${getProviderInfo(formProvider).name}...`
    )

    try {
      const payload = {
        provider: formProvider,
        email: formEmail.trim(),
        password: formPassword.trim(),
        token: formToken.trim(),
        cookies: formCookies.trim(),
        username: formUsername.trim(),
      }

      const res = await fetch(`${API_BASE}/api/admin/accounts`, {
        method: "POST",
        headers: {
          ...getAuthHeader(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      })

      const data = await res.json()
      if (!res.ok || data.ok === false) {
        throw new Error(data.error || "Gagal menyimpan akun")
      }

      toast.success(`Akun ${data.email || formEmail} berhasil ditambahkan!`, { id: toastId })
      setModalOpen(false)
      fetchAccounts()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Terjadi kesalahan saat menyimpan akun"
      toast.error(msg, { id: toastId })
    } finally {
      setSubmitting(false)
    }
  }

  const handleVerifyAccount = async (email: string, provider?: string) => {
    const prov = provider || "deepseek"
    setVerifyingEmail(`${email}:${prov}`)
    const toastId = toast.loading(`Memverifikasi akun ${email} (${prov})...`)
    try {
      const provParam = `?provider=${encodeURIComponent(prov)}`
      const res = await fetch(`${API_BASE}/api/admin/accounts/${encodeURIComponent(email)}/verify${provParam}`, {
        method: "POST",
        headers: getAuthHeader(),
      })
      const data = await res.json()
      if (!res.ok || data.valid === false) {
        throw new Error(data.error || "Verifikasi token gagal")
      }
      toast.success(`Token akun ${email} (${prov}) valid dan aktif!`, { id: toastId })
      fetchAccounts()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Verifikasi token gagal"
      toast.error(msg, { id: toastId })
    } finally {
      setVerifyingEmail(null)
    }
  }

  const handleReLogin = async (email: string, provider?: string) => {
    const prov = provider || "deepseek"
    const toastId = toast.loading(`Login ulang untuk ${email} (${prov})...`)
    try {
      const provParam = `?provider=${encodeURIComponent(prov)}`
      const res = await fetch(`${API_BASE}/api/admin/accounts/${encodeURIComponent(email)}/activate${provParam}`, {
        method: "POST",
        headers: getAuthHeader(),
      })
      const data = await res.json()
      if (!res.ok || data.ok === false) {
        throw new Error(data.error || data.message || "Gagal login ulang")
      }
      toast.success(data.message || `Login ulang ${email} (${prov}) berhasil!`, { id: toastId })
      fetchAccounts()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal login ulang"
      toast.error(msg, { id: toastId })
    }
  }

  const handleDeleteAccount = async (email: string, provider?: string) => {
    const prov = provider || "deepseek"
    if (!confirm(`Hapus akun ${email} (${prov})?`)) return
    try {
      const provParam = `?provider=${encodeURIComponent(prov)}`
      const res = await fetch(`${API_BASE}/api/admin/accounts/${encodeURIComponent(email)}${provParam}`, {
        method: "DELETE",
        headers: getAuthHeader(),
      })
      if (!res.ok) throw new Error("Gagal menghapus akun")
      toast.success(`Akun ${email} (${prov}) berhasil dihapus`)
      setAccounts(accounts.filter(a => !(a.email === email && (a.provider || "deepseek") === prov)))
    } catch {
      toast.error("Gagal menghapus akun")
    }
  }

  // Filtered Accounts
  const filteredAccounts = useMemo(() => {
    return accounts.filter(acc => {
      const prov = acc.provider || "deepseek"
      if (selectedProviderTab !== "all" && prov !== selectedProviderTab) return false
      if (statusFilter !== "all") {
        if (statusFilter === "valid" && !acc.valid) return false
        if (statusFilter === "rate_limited" && acc.status_code !== "rate_limited") return false
        if (statusFilter === "invalid" && acc.valid) return false
      }
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase()
        const match =
          acc.email.toLowerCase().includes(q) ||
          (acc.username || "").toLowerCase().includes(q) ||
          (acc.token || "").toLowerCase().includes(q) ||
          (acc.last_error || "").toLowerCase().includes(q)
        if (!match) return false
      }
      return true
    })
  }, [accounts, selectedProviderTab, statusFilter, searchQuery])

  // Stats calculation
  const stats = useMemo(() => {
    const list = selectedProviderTab === "all" ? accounts : accounts.filter(a => (a.provider || "deepseek") === selectedProviderTab)
    const valid = list.filter(a => a.valid).length
    const rateLimited = list.filter(a => a.status_code === "rate_limited").length
    const invalid = list.length - valid
    return { total: list.length, valid, rateLimited, invalid }
  }, [accounts, selectedProviderTab])

  return (
    <div className="space-y-6">
      {/* Header */}
      <section className="relative overflow-hidden rounded-[32px] border border-white/75 bg-card/82 p-6 shadow-[var(--shadow-lift)] backdrop-blur-sm">
        <div className="pointer-events-none absolute -right-20 -top-24 size-64 rounded-full bg-primary/20 blur-3xl" />
        <div className="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div className="text-xs font-black uppercase tracking-[0.28em] text-muted-foreground">Pool Kredensial Multi-Provider</div>
            <h2 className="mt-2 text-4xl font-black tracking-tight">Manajemen Akun Provider</h2>
            <p className="mt-2 max-w-3xl text-muted-foreground leading-relaxed">
              Kelola kumpulan akun untuk <strong>DeepSeek</strong>, <strong>Qwen</strong>, <strong>ChatGPT</strong>, dan <strong>Mistral</strong>.
              Sistem akan merotasi akun aktif secara otomatis untuk menjaga kelancaran request tanpa batas kuota.
            </p>
          </div>
          <div className="flex flex-wrap gap-2.5">
            <Button
              onClick={() => openAddModal(selectedProviderTab === "all" ? "deepseek" : selectedProviderTab)}
              className="flex items-center gap-2 rounded-2xl bg-primary px-5 py-3 font-bold text-primary-foreground shadow-lg shadow-primary/25 transition hover:opacity-95"
            >
              <Plus className="size-5" />
              <span>Tambah Akun Baru</span>
            </Button>
            <Button
              variant="outline"
              onClick={fetchAccounts}
              className="rounded-2xl flex items-center gap-1.5 font-bold"
            >
              <RefreshCw className="size-4" />
              <span>Segarkan</span>
            </Button>
          </div>
        </div>
      </section>

      {/* Provider Selector Tabs */}
      <div className="flex flex-wrap items-center gap-2 border-b border-border/50 pb-3">
        {PROVIDERS.map(p => {
          const count = p.id === "all" ? accounts.length : accounts.filter(a => (a.provider || "deepseek") === p.id).length
          const active = selectedProviderTab === p.id
          return (
            <button
              key={p.id}
              onClick={() => setSelectedProviderTab(p.id)}
              className={`flex items-center gap-2 rounded-2xl border px-4 py-2.5 text-xs font-bold transition ${
                active
                  ? "border-primary bg-primary text-primary-foreground shadow-md shadow-primary/20"
                  : "border-border/60 bg-card/70 text-muted-foreground hover:bg-muted/40 hover:text-foreground"
              }`}
            >
              <span>{p.name}</span>
              <span
                className={`rounded-full px-2 py-0.5 text-[11px] font-black ${
                  active ? "bg-primary-foreground/20 text-primary-foreground" : "bg-muted text-foreground/80"
                }`}
              >
                {count}
              </span>
            </button>
          )
        })}
      </div>

      {/* Stat Cards Row */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-[28px] border border-white/75 bg-card/80 p-5 shadow-sm">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-bold uppercase tracking-wider">Total Akun ({selectedProviderTab === "all" ? "Semua" : getProviderInfo(selectedProviderTab).name})</span>
            <Layers className="size-4" />
          </div>
          <div className="mt-3 text-3xl font-black">{stats.total}</div>
        </div>
        <div className="rounded-[28px] border border-white/75 bg-card/80 p-5 shadow-sm">
          <div className="flex items-center justify-between text-emerald-600">
            <span className="text-xs font-bold uppercase tracking-wider">Akun Normal / Aktif</span>
            <CheckCircle2 className="size-4" />
          </div>
          <div className="mt-3 text-3xl font-black text-emerald-600">{stats.valid}</div>
        </div>
        <div className="rounded-[28px] border border-white/75 bg-card/80 p-5 shadow-sm">
          <div className="flex items-center justify-between text-amber-600">
            <span className="text-xs font-bold uppercase tracking-wider">Terkena Rate Limit</span>
            <ShieldAlert className="size-4" />
          </div>
          <div className="mt-3 text-3xl font-black text-amber-600">{stats.rateLimited}</div>
        </div>
        <div className="rounded-[28px] border border-white/75 bg-card/80 p-5 shadow-sm">
          <div className="flex items-center justify-between text-rose-600">
            <span className="text-xs font-bold uppercase tracking-wider">Gagal / Butuh Relogin</span>
            <XCircle className="size-4" />
          </div>
          <div className="mt-3 text-3xl font-black text-rose-600">{stats.invalid}</div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            placeholder="Cari email, token, username..."
            className="w-full rounded-2xl border border-border/60 bg-card/80 py-2.5 pl-10 pr-4 text-xs font-medium focus:outline-none focus:ring-2 focus:ring-primary/40"
          />
        </div>
        <div className="flex items-center gap-2">
          <select
            value={statusFilter}
            onChange={e => setStatusFilter(e.target.value)}
            className="rounded-2xl border border-border/60 bg-card/80 px-3.5 py-2.5 text-xs font-bold focus:outline-none focus:ring-2 focus:ring-primary/40"
          >
            <option value="all">Semua Status</option>
            <option value="valid">Hanya Normal (Valid)</option>
            <option value="rate_limited">Hanya Rate Limited</option>
            <option value="invalid">Hanya Gagal/Invalid</option>
          </select>
        </div>
      </div>

      {/* Accounts List Table */}
      <section className="overflow-hidden rounded-[30px] border border-white/75 bg-card/86 shadow-[var(--shadow-lift)]">
        <div className="border-b border-border/50 bg-muted/10 px-6 py-4 flex items-center justify-between">
          <div>
            <h3 className="text-lg font-black tracking-tight">Daftar Akun Provider</h3>
            <p className="text-xs text-muted-foreground">Menampilkan {filteredAccounts.length} dari {accounts.length} akun terdaftar</p>
          </div>
        </div>

        {loading ? (
          <div className="flex h-48 items-center justify-center">
            <div className="flex items-center gap-3 text-muted-foreground">
              <RefreshCw className="size-5 animate-spin" />
              <span>Memuat data akun...</span>
            </div>
          </div>
        ) : filteredAccounts.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-12 text-center">
            <Activity className="size-12 text-muted-foreground/40 mb-3" />
            <h4 className="font-bold text-base">Tidak Ada Akun Ditemukan</h4>
            <p className="text-xs text-muted-foreground mt-1 max-w-sm">
              Belum ada akun untuk filter ini. Silakan klik tombol "Tambah Akun Baru" di atas.
            </p>
            <Button onClick={() => openAddModal(selectedProviderTab === "all" ? "deepseek" : selectedProviderTab)} className="mt-4 rounded-xl text-xs">
              Tambah Akun {selectedProviderTab === "all" ? "DeepSeek" : getProviderInfo(selectedProviderTab).name}
            </Button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[840px] text-left text-sm">
              <thead className="border-b bg-muted/25 text-[11px] uppercase tracking-wider text-muted-foreground">
                <tr>
                  <th className="px-6 py-3 font-bold">Provider</th>
                  <th className="px-4 py-3 font-bold">Email / Akun</th>
                  <th className="px-4 py-3 font-bold">Status</th>
                  <th className="px-4 py-3 font-bold">Token Sesi</th>
                  <th className="px-4 py-3 text-right font-bold">In-flight</th>
                  <th className="px-6 py-3 text-right font-bold">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/50">
                {filteredAccounts.map(acc => {
                  const pInfo = getProviderInfo(acc.provider)
                  return (
                    <tr key={acc.email} className="transition hover:bg-black/5 dark:hover:bg-white/5">
                      <td className="px-6 py-4">
                        <span className={`inline-flex items-center rounded-xl border px-2.5 py-1 text-xs font-bold ${pInfo.color}`}>
                          {pInfo.name}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <div className="font-mono text-xs font-bold text-foreground">{acc.email}</div>
                        {acc.username && <div className="text-[11px] text-muted-foreground">User: {acc.username}</div>}
                        {acc.last_error && <div className="text-[11px] text-rose-500 line-clamp-1 mt-0.5">{acc.last_error}</div>}
                      </td>
                      <td className="px-4 py-4">
                        <span
                          className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-bold ring-1 ${
                            acc.valid
                              ? "bg-emerald-500/10 text-emerald-700 ring-emerald-500/30 dark:text-emerald-400"
                              : acc.status_code === "rate_limited"
                              ? "bg-amber-500/10 text-amber-700 ring-amber-500/30 dark:text-amber-400"
                              : "bg-rose-500/10 text-rose-700 ring-rose-500/30 dark:text-rose-400"
                          }`}
                        >
                          {acc.valid ? "Normal" : acc.status_code === "rate_limited" ? "Rate Limited" : "Gagal / Banned"}
                        </span>
                      </td>
                      <td className="px-4 py-4 font-mono text-xs text-muted-foreground">
                        <div className="flex items-center gap-1.5">
                          <span>{maskedToken(acc.token)}</span>
                          {acc.token && (
                            <button
                              type="button"
                              onClick={() => {
                                navigator.clipboard.writeText(acc.token || "")
                                toast.success("Token disalin")
                              }}
                              className="rounded p-1 hover:bg-muted text-muted-foreground"
                              title="Salin Token"
                            >
                              <Copy className="size-3.5" />
                            </button>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-4 text-right font-mono text-xs font-bold">
                        {acc.inflight || 0}<span className="text-muted-foreground">/{acc.max_inflight || 4}</span>
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={verifyingEmail === `${acc.email}:${acc.provider || "deepseek"}`}
                            onClick={() => handleVerifyAccount(acc.email, acc.provider)}
                            className="rounded-xl text-xs font-bold px-2.5 h-8"
                          >
                            {verifyingEmail === `${acc.email}:${acc.provider || "deepseek"}` ? (
                              <RefreshCw className="size-3 animate-spin" />
                            ) : (
                              "Verifikasi"
                            )}
                          </Button>
                          {acc.password && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleReLogin(acc.email, acc.provider)}
                              className="rounded-xl text-xs font-bold px-2.5 h-8 text-blue-600 hover:text-blue-700"
                              title="Login ulang via kredensial tersimpan"
                            >
                              <LogIn className="size-3" />
                            </Button>
                          )}
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleDeleteAccount(acc.email, acc.provider)}
                            className="rounded-xl text-xs font-bold px-2.5 h-8 text-rose-600 hover:text-rose-700"
                            title="Hapus akun"
                          >
                            <Trash2 className="size-3.5" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {/* Modal Tambah Akun Multi-Provider */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
          <div className="relative max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-[32px] border border-white/80 bg-card p-6 shadow-2xl">
            <div className="flex items-center justify-between border-b border-border/50 pb-4">
              <div className="flex items-center gap-2.5">
                <div className="rounded-xl bg-primary/10 p-2 text-primary">
                  <Plus className="size-5" />
                </div>
                <h3 className="text-xl font-black">Tambah Akun Provider</h3>
              </div>
              <button
                onClick={() => setModalOpen(false)}
                className="rounded-full p-2 text-muted-foreground hover:bg-muted"
              >
                <X className="size-5" />
              </button>
            </div>

            <div className="mt-5 space-y-4 text-sm">
              {/* Step 1: Pilih Provider */}
              <div>
                <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-2">
                  1. Pilih AI Provider
                </label>
                <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
                  {PROVIDERS.filter(p => p.id !== "all").map(p => {
                    const active = formProvider === p.id
                    return (
                      <div
                        key={p.id}
                        onClick={() => handleProviderSelectInModal(p.id)}
                        className={`cursor-pointer rounded-2xl border p-3 text-center transition ${
                          active
                            ? "border-primary bg-primary/10 ring-2 ring-primary text-foreground font-black"
                            : "border-border/60 bg-muted/10 hover:bg-muted/30 text-muted-foreground font-bold"
                        }`}
                      >
                        <div className="text-xs">{p.name}</div>
                      </div>
                    )
                  })}
                </div>
              </div>

              {/* Step 2: Mode Input (Auto Login vs Token) */}
              {(formProvider === "deepseek" || formProvider === "qwen") && (
                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                    2. Metode Autentikasi
                  </label>
                  <div className="grid grid-cols-2 gap-2.5">
                    <div
                      onClick={() => setLoginMode("auto")}
                      className={`cursor-pointer rounded-xl border p-3 transition ${
                        loginMode === "auto"
                          ? "border-primary bg-primary/5 ring-1 ring-primary font-bold text-foreground"
                          : "border-border/60 bg-muted/10 text-muted-foreground"
                      }`}
                    >
                      <div className="flex items-center gap-1.5 text-xs">
                        <LogIn className="size-3.5" />
                        <span>Login Otomatis (Email & Sandi)</span>
                      </div>
                    </div>

                    <div
                      onClick={() => setLoginMode("token")}
                      className={`cursor-pointer rounded-xl border p-3 transition ${
                        loginMode === "token"
                          ? "border-primary bg-primary/5 ring-1 ring-primary font-bold text-foreground"
                          : "border-border/60 bg-muted/10 text-muted-foreground"
                      }`}
                    >
                      <div className="flex items-center gap-1.5 text-xs">
                        <Key className="size-3.5" />
                        <span>Input Token Manual</span>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Form Input Fields */}
              <div className="rounded-2xl border border-border/50 bg-muted/20 p-4 space-y-3.5">
                {loginMode === "auto" ? (
                  <>
                    <div>
                      <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                        Email Akun {getProviderInfo(formProvider).name}
                      </label>
                      <input
                        type="email"
                        value={formEmail}
                        onChange={e => setFormEmail(e.target.value)}
                        placeholder="user@example.com"
                        className="w-full rounded-xl border border-border/60 bg-card px-3.5 py-2.5 text-xs focus:outline-none focus:ring-2 focus:ring-primary/40"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                        Kata Sandi
                      </label>
                      <input
                        type="password"
                        value={formPassword}
                        onChange={e => setFormPassword(e.target.value)}
                        placeholder="••••••••••••"
                        className="w-full rounded-xl border border-border/60 bg-card px-3.5 py-2.5 text-xs focus:outline-none focus:ring-2 focus:ring-primary/40"
                      />
                    </div>
                    <div className="text-[11px] text-muted-foreground leading-relaxed">
                      {formProvider === "deepseek"
                        ? "Sistem akan otomatis menghitung tantangan Wasm SHA-3 Proof-of-Work dan mengambil token sesi resmi Android DeepSeek."
                        : "Sistem akan melakukan login langsung ke platform Alibaba Cloud Qwen dan menyimpan token serta cookies sesi."}
                    </div>
                  </>
                ) : (
                  <>
                    <div>
                      <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                        Label / Email Akun (Opsional)
                      </label>
                      <input
                        type="text"
                        value={formEmail}
                        onChange={e => setFormEmail(e.target.value)}
                        placeholder={`${formProvider}_account_1`}
                        className="w-full rounded-xl border border-border/60 bg-card px-3.5 py-2.5 text-xs focus:outline-none focus:ring-2 focus:ring-primary/40"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                        {formProvider === "chatgpt"
                          ? "Session Token / Bearer Token"
                          : formProvider === "qwen"
                          ? "Token JWT Qwen"
                          : "Token Sesi / API Key"}
                      </label>
                      <textarea
                        rows={3}
                        value={formToken}
                        onChange={e => setFormToken(e.target.value)}
                        placeholder={
                          formProvider === "deepseek"
                            ? "wqOrGxK+OJKF..."
                            : formProvider === "chatgpt"
                            ? "Bearer eyJhbGci..."
                            : "eyJhbGciOiJIUzI1NiIs..."
                        }
                        className="w-full rounded-xl border border-border/60 bg-card px-3.5 py-2 font-mono text-xs focus:outline-none focus:ring-2 focus:ring-primary/40"
                      />
                    </div>
                    {(formProvider === "qwen" || formProvider === "chatgpt") && (
                      <div>
                        <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                          Cookies (Opsional, disarankan untuk bypass WAF)
                        </label>
                        <textarea
                          rows={2}
                          value={formCookies}
                          onChange={e => setFormCookies(e.target.value)}
                          placeholder="token=eyJ...; acw_tc=0a03..."
                          className="w-full rounded-xl border border-border/60 bg-card px-3.5 py-2 font-mono text-xs focus:outline-none focus:ring-2 focus:ring-primary/40"
                        />
                      </div>
                    )}
                  </>
                )}
              </div>

              {/* Action Buttons */}
              <div className="flex items-center justify-end gap-3 border-t border-border/50 pt-4">
                <Button variant="outline" onClick={() => setModalOpen(false)} className="rounded-xl text-xs">
                  Batal
                </Button>
                <Button
                  disabled={submitting}
                  onClick={handleSaveAccount}
                  className="rounded-xl bg-primary text-primary-foreground font-bold text-xs"
                >
                  {submitting ? "Memproses..." : "Simpan Akun"}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
