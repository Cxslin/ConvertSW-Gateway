import { useEffect, useState } from "react"
import {
  Layers,
  Plus,
  Trash2,
  Edit2,
  Play,
  ArrowRight,
  RefreshCw,
  ShieldCheck,
  Check,
  X,
  MoveUp,
  MoveDown,
} from "lucide-react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"
import { Button } from "../components/ui/button"
import { getAuthHeader } from "../lib/auth"
import { API_BASE } from "../lib/api"

export type ComboModel = {
  id: string
  name: string
  description: string
  strategy: "fallback" | "round_robin"
  targets: string[]
  enabled: boolean
}

const PROVIDER_OPTIONS = [
  {
    id: "deepseek",
    name: "DeepSeek AI",
    color: "bg-blue-500/10 text-blue-700 border-blue-500/30 dark:text-blue-400",
    models: [
      { id: "deepseek-reasoner", label: "DeepSeek-R1 (Reasoner)" },
      { id: "deepseek-chat", label: "DeepSeek-V3 (Chat)" },
      { id: "deepseek-v3", label: "DeepSeek-V3" },
      { id: "deepseek-r1", label: "DeepSeek-R1" },
      { id: "deepseek-vision", label: "DeepSeek-Vision" },
    ],
  },
  {
    id: "chatgpt",
    name: "ChatGPT (OpenAI)",
    color: "bg-emerald-500/10 text-emerald-700 border-emerald-500/30 dark:text-emerald-400",
    models: [
      { id: "gpt-5-6", label: "ChatGPT 5/4o (Auto)" },
      { id: "gpt-4o", label: "GPT-4o (Omni)" },
      { id: "gpt-4o-mini", label: "GPT-4o Mini (Fast)" },
      { id: "chatgpt-auto", label: "ChatGPT Auto" },
      { id: "chatgpt-search", label: "ChatGPT Search" },
    ],
  },
  {
    id: "mistral",
    name: "Mistral AI",
    color: "bg-amber-500/10 text-amber-700 border-amber-500/30 dark:text-amber-400",
    models: [
      { id: "mistral-chat", label: "Mistral Chat (Fast)" },
      { id: "mistral-large", label: "Mistral Large (Reasoning)" },
      { id: "mistral-small", label: "Mistral Small" },
      { id: "mistral-vibe", label: "Mistral Codestral/Vibe" },
      { id: "mistral-search", label: "Mistral Search" },
    ],
  },
  {
    id: "qwen",
    name: "Qwen AI (Alibaba)",
    color: "bg-purple-500/10 text-purple-700 border-purple-500/30 dark:text-purple-400",
    models: [
      { id: "qwen3.8-max", label: "Qwen 3.8 Max (Flagship)" },
      { id: "qwen3.7-max", label: "Qwen 3.7 Max" },
      { id: "qwen3.7-plus", label: "Qwen 3.7 Plus (Balanced)" },
      { id: "qwen3.6-plus", label: "Qwen 3.6 Plus (Agent/Code)" },
      { id: "qwen3.5-plus", label: "Qwen 3.5 Plus (Fast)" },
      { id: "qwen3.5-omni-plus", label: "Qwen 3.5 Omni (Multimodal)" },
      { id: "qwen-max", label: "Qwen Max (Alias -> 3.8 Max)" },
      { id: "qwen-plus", label: "Qwen Plus (Alias -> 3.7 Plus)" },
      { id: "qwen-turbo", label: "Qwen Turbo (Alias -> 3.5 Plus)" },
      { id: "qwen2.5-72b-instruct", label: "Qwen 2.5 72B Instruct" },
    ],
  },
]

export default function ComboModelsPage() {
  const navigate = useNavigate()
  const [combos, setCombos] = useState<ComboModel[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingCombo, setEditingCombo] = useState<ComboModel | null>(null)

  // Form states
  const [formId, setFormId] = useState("")
  const [formName, setFormName] = useState("")
  const [formDesc, setFormDesc] = useState("")
  const [formStrategy, setFormStrategy] = useState<"fallback" | "round_robin">("fallback")
  const [formTargets, setFormTargets] = useState<string[]>([])
  const [selectedProvider, setSelectedProvider] = useState(PROVIDER_OPTIONS[0].id)
  const [selectedModel, setSelectedModel] = useState(PROVIDER_OPTIONS[0].models[0].id)

  const fetchCombos = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/combos`, { headers: getAuthHeader() })
      if (!res.ok) throw new Error("Gagal memuat combo model")
      const data = await res.json()
      if (Array.isArray(data.combos)) {
        setCombos(data.combos)
      }
    } catch {
      toast.error("Gagal mengambil data combo model dari server")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchCombos()
  }, [])

  const openCreateModal = () => {
    setEditingCombo(null)
    setFormId("")
    setFormName("")
    setFormDesc("")
    setFormStrategy("fallback")
    setFormTargets([
      "deepseek:deepseek-reasoner",
      "chatgpt:gpt-5-6",
      "mistral:mistral-large",
    ])
    setSelectedProvider(PROVIDER_OPTIONS[0].id)
    setSelectedModel(PROVIDER_OPTIONS[0].models[0].id)
    setModalOpen(true)
  }

  const openEditModal = (c: ComboModel) => {
    setEditingCombo(c)
    setFormId(c.id)
    setFormName(c.name)
    setFormDesc(c.description)
    setFormStrategy(c.strategy || "fallback")
    setFormTargets([...c.targets])
    setSelectedProvider(PROVIDER_OPTIONS[0].id)
    setSelectedModel(PROVIDER_OPTIONS[0].models[0].id)
    setModalOpen(true)
  }

  const handleProviderChange = (provId: string) => {
    setSelectedProvider(provId)
    const prov = PROVIDER_OPTIONS.find(p => p.id === provId)
    if (prov && prov.models.length > 0) {
      setSelectedModel(prov.models[0].id)
    }
  }

  const handleAddTarget = () => {
    const targetKey = `${selectedProvider}:${selectedModel}`
    if (formTargets.includes(targetKey)) {
      toast.info("Target model sudah ada dalam rantai")
      return
    }
    setFormTargets([...formTargets, targetKey])
  }

  const handleRemoveTarget = (index: number) => {
    setFormTargets(formTargets.filter((_, i) => i !== index))
  }

  const handleMoveTarget = (index: number, direction: "up" | "down") => {
    const newIdx = direction === "up" ? index - 1 : index + 1
    if (newIdx < 0 || newIdx >= formTargets.length) return
    const updated = [...formTargets]
    const temp = updated[index]
    updated[index] = updated[newIdx]
    updated[newIdx] = temp
    setFormTargets(updated)
  }

  const handleSaveCombo = async () => {
    const cleanId = formId.trim().toLowerCase().replace(/[^a-z0-9_-]/g, "-")
    if (!cleanId) {
      toast.error("ID Combo model wajib diisi")
      return
    }
    if (!cleanId.startsWith("combo-")) {
      toast.error("ID Combo harus diawali dengan 'combo-' (misal: combo-custom)")
      return
    }
    if (!formName.trim()) {
      toast.error("Nama Combo wajib diisi")
      return
    }
    if (formTargets.length === 0) {
      toast.error("Tambahkan minimal 1 target provider:model")
      return
    }

    const payload: ComboModel = {
      id: cleanId,
      name: formName.trim(),
      description: formDesc.trim(),
      strategy: formStrategy,
      targets: formTargets,
      enabled: editingCombo ? editingCombo.enabled : true,
    }

    try {
      const res = await fetch(`${API_BASE}/api/admin/combos`, {
        method: "POST",
        headers: {
          ...getAuthHeader(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || "Gagal menyimpan combo")
      }
      toast.success(editingCombo ? "Combo model berhasil diperbarui" : "Combo model baru berhasil dibuat")
      setModalOpen(false)
      fetchCombos()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal menyimpan combo"
      toast.error(msg)
    }
  }

  const handleToggleEnable = async (combo: ComboModel) => {
    const updated: ComboModel = { ...combo, enabled: !combo.enabled }
    try {
      const res = await fetch(`${API_BASE}/api/admin/combos`, {
        method: "POST",
        headers: {
          ...getAuthHeader(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify(updated),
      })
      if (!res.ok) throw new Error("Gagal mengubah status")
      toast.success(`Combo ${combo.id} ${updated.enabled ? "diaktifkan" : "dinonaktifkan"}`)
      setCombos(combos.map(c => (c.id === combo.id ? updated : c)))
    } catch {
      toast.error("Gagal mengubah status combo model")
    }
  }

  const handleDeleteCombo = async (id: string) => {
    if (!confirm(`Yakin ingin menghapus combo '${id}'?`)) return
    try {
      const res = await fetch(`${API_BASE}/api/admin/combos/${encodeURIComponent(id)}`, {
        method: "DELETE",
        headers: getAuthHeader(),
      })
      if (!res.ok) throw new Error("Gagal menghapus")
      toast.success(`Combo ${id} berhasil dihapus`)
      setCombos(combos.filter(c => c.id !== id))
    } catch {
      toast.error("Gagal menghapus combo model")
    }
  }

  const getTargetBadge = (targetStr: string) => {
    const [prov, model] = targetStr.split(":")
    const provInfo = PROVIDER_OPTIONS.find(p => p.id === prov)
    return {
      provName: provInfo ? provInfo.name : prov,
      modelName: model || prov,
      color: provInfo ? provInfo.color : "bg-muted text-muted-foreground",
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <section className="relative overflow-hidden rounded-[32px] border border-white/75 bg-card/82 p-6 shadow-[var(--shadow-lift)] backdrop-blur-sm">
        <div className="pointer-events-none absolute -right-20 -top-24 size-64 rounded-full bg-primary/20 blur-3xl" />
        <div className="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div className="text-xs font-black uppercase tracking-[0.28em] text-muted-foreground">Multi-Provider Routing</div>
            <h2 className="mt-2 text-4xl font-black tracking-tight">Model Combo & Auto-Rolling</h2>
            <p className="mt-2 max-w-3xl text-muted-foreground leading-relaxed">
              Kombinasikan beberapa provider (DeepSeek, ChatGPT, Mistral, Qwen) ke dalam satu virtual model tunggal.
              Didukung <strong>Failover Otomatis</strong> saat rate-limit/error atau <strong>Round-Robin Rolling</strong> bergantian untuk distribusi beban.
            </p>
          </div>
          <Button
            onClick={openCreateModal}
            className="flex items-center gap-2 rounded-2xl bg-primary px-5 py-3 font-bold text-primary-foreground shadow-lg shadow-primary/25 transition hover:opacity-95"
          >
            <Plus className="size-5" />
            <span>Tambah Combo Baru</span>
          </Button>
        </div>
      </section>

      {/* Overview Info Banner */}
      <div className="grid gap-4 md:grid-cols-2">
        <div className="flex items-start gap-3 rounded-[24px] border border-white/75 bg-card/75 p-5 shadow-sm">
          <div className="rounded-xl bg-primary/10 p-2 text-primary">
            <ShieldCheck className="size-5" />
          </div>
          <div>
            <h4 className="font-bold text-foreground">Strategi Smart Fallback</h4>
            <p className="mt-1 text-xs text-muted-foreground leading-relaxed">
              Memanggil target urutan ke-1 terlebih dahulu. Jika terjadi HTTP error (429 rate limit, 500 server error, timeout), router secara transparan mengalihkan permintaan ke target berikutnya tanpa memutus koneksi client.
            </p>
          </div>
        </div>

        <div className="flex items-start gap-3 rounded-[24px] border border-white/75 bg-card/75 p-5 shadow-sm">
          <div className="rounded-xl bg-accent/20 p-2 text-accent-foreground">
            <RefreshCw className="size-5" />
          </div>
          <div>
            <h4 className="font-bold text-foreground">Strategi Round-Robin Rolling</h4>
            <p className="mt-1 text-xs text-muted-foreground leading-relaxed">
              Memutar target secara bergantian pada setiap request baru (Target 1 &rarr; Target 2 &rarr; Target 3 &rarr; Target 1). Sangat cocok untuk menghemat kuota rate limit dan mendistribusikan beban secara merata antar provider.
            </p>
          </div>
        </div>
      </div>

      {/* Combos List */}
      <div className="space-y-4">
        {loading ? (
          <div className="flex h-48 items-center justify-center rounded-[30px] border border-white/75 bg-card/86">
            <div className="flex items-center gap-3 text-muted-foreground">
              <RefreshCw className="size-5 animate-spin" />
              <span>Memuat daftar combo model...</span>
            </div>
          </div>
        ) : combos.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-[30px] border border-white/75 bg-card/86 p-12 text-center">
            <Layers className="size-12 text-muted-foreground/50 mb-3" />
            <h3 className="text-lg font-bold">Belum Ada Model Combo</h3>
            <p className="text-sm text-muted-foreground mt-1 max-w-sm">
              Buat model virtual pertama Anda untuk menggabungkan beberapa provider dengan mekanisme fallback cerdas.
            </p>
            <Button onClick={openCreateModal} className="mt-4 rounded-xl">
              Buat Combo Baru
            </Button>
          </div>
        ) : (
          combos.map(combo => (
            <div
              key={combo.id}
              className={`rounded-[30px] border border-white/75 bg-card/86 p-6 shadow-[var(--shadow-lift)] transition ${
                !combo.enabled ? "opacity-60 grayscale-[40%]" : ""
              }`}
            >
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-1">
                  <div className="flex flex-wrap items-center gap-2.5">
                    <span className="font-mono text-xs font-black uppercase tracking-wider rounded-lg bg-muted px-2.5 py-1 text-foreground/80">
                      {combo.id}
                    </span>
                    <span
                      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-bold ring-1 ${
                        combo.strategy === "round_robin"
                          ? "bg-purple-500/10 text-purple-700 ring-purple-500/30 dark:text-purple-400"
                          : "bg-blue-500/10 text-blue-700 ring-blue-500/30 dark:text-blue-400"
                      }`}
                    >
                      {combo.strategy === "round_robin" ? (
                        <>
                          <RefreshCw className="size-3" />
                          <span>Round Robin (Rolling)</span>
                        </>
                      ) : (
                        <>
                          <ArrowRight className="size-3" />
                          <span>Smart Fallback (Failover)</span>
                        </>
                      )}
                    </span>
                    <span
                      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-bold ring-1 ${
                        combo.enabled
                          ? "bg-emerald-500/10 text-emerald-700 ring-emerald-500/30 dark:text-emerald-400"
                          : "bg-stone-500/10 text-stone-600 ring-stone-500/30"
                      }`}
                    >
                      {combo.enabled ? "Aktif" : "Nonaktif"}
                    </span>
                  </div>
                  <h3 className="text-xl font-black tracking-tight text-foreground pt-1">{combo.name}</h3>
                  <p className="text-sm text-muted-foreground">{combo.description}</p>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate(`/test?model=${combo.id}`)}
                    className="rounded-xl flex items-center gap-1.5 text-xs font-bold"
                  >
                    <Play className="size-3.5 text-emerald-600" />
                    <span>Uji Chat</span>
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleToggleEnable(combo)}
                    className="rounded-xl text-xs font-bold"
                  >
                    {combo.enabled ? "Nonaktifkan" : "Aktifkan"}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => openEditModal(combo)}
                    className="rounded-xl flex items-center gap-1.5 text-xs font-bold"
                  >
                    <Edit2 className="size-3.5" />
                    <span>Edit</span>
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleDeleteCombo(combo.id)}
                    className="rounded-xl flex items-center gap-1.5 text-xs font-bold text-rose-600 hover:text-rose-700"
                  >
                    <Trash2 className="size-3.5" />
                    <span>Hapus</span>
                  </Button>
                </div>
              </div>

              {/* Target Chain Flow Visualizer */}
              <div className="mt-5 rounded-2xl border border-border/40 bg-muted/20 p-4">
                <div className="text-xs font-bold text-muted-foreground uppercase tracking-wider mb-2.5">
                  Rantai Eksekusi Target ({combo.targets.length} Target)
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {combo.targets.map((target, idx) => {
                    const info = getTargetBadge(target)
                    return (
                      <div key={idx} className="flex items-center gap-2">
                        <div className={`flex items-center gap-1.5 rounded-xl border px-3 py-1.5 text-xs font-bold shadow-sm ${info.color}`}>
                          <span className="flex size-4 items-center justify-center rounded-full bg-black/10 dark:bg-white/10 text-[10px]">
                            {idx + 1}
                          </span>
                          <span className="font-semibold">{info.provName}:</span>
                          <span className="font-mono">{info.modelName}</span>
                        </div>
                        {idx < combo.targets.length - 1 && (
                          <ArrowRight className="size-3.5 text-muted-foreground/60 shrink-0" />
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Modal Buat / Edit Combo */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
          <div className="relative max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-[32px] border border-white/80 bg-card p-6 shadow-2xl">
            <div className="flex items-center justify-between border-b border-border/50 pb-4">
              <div className="flex items-center gap-2.5">
                <div className="rounded-xl bg-primary/10 p-2 text-primary">
                  <Layers className="size-5" />
                </div>
                <h3 className="text-xl font-black">
                  {editingCombo ? "Edit Combo Model" : "Buat Combo Model Baru"}
                </h3>
              </div>
              <button
                onClick={() => setModalOpen(false)}
                className="rounded-full p-2 text-muted-foreground hover:bg-muted"
              >
                <X className="size-5" />
              </button>
            </div>

            <div className="mt-5 space-y-4 text-sm">
              {/* ID & Name */}
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                    Model ID (Harus berawalan 'combo-')
                  </label>
                  <input
                    type="text"
                    value={formId}
                    onChange={e => setFormId(e.target.value)}
                    disabled={Boolean(editingCombo)}
                    placeholder="combo-smart"
                    className="w-full rounded-xl border border-border/60 bg-muted/20 px-3.5 py-2.5 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-primary/40 disabled:opacity-60"
                  />
                </div>
                <div>
                  <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                    Nama Tampilan
                  </label>
                  <input
                    type="text"
                    value={formName}
                    onChange={e => setFormName(e.target.value)}
                    placeholder="Combo Smart Router"
                    className="w-full rounded-xl border border-border/60 bg-muted/20 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/40"
                  />
                </div>
              </div>

              {/* Description */}
              <div>
                <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Deskripsi
                </label>
                <input
                  type="text"
                  value={formDesc}
                  onChange={e => setFormDesc(e.target.value)}
                  placeholder="Keterangan singkat fungsi combo ini..."
                  className="w-full rounded-xl border border-border/60 bg-muted/20 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/40"
                />
              </div>

              {/* Strategy */}
              <div>
                <label className="block text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Strategi Distribusi
                </label>
                <div className="grid gap-3 sm:grid-cols-2">
                  <div
                    onClick={() => setFormStrategy("fallback")}
                    className={`cursor-pointer rounded-2xl border p-4 transition ${
                      formStrategy === "fallback"
                        ? "border-primary bg-primary/5 ring-1 ring-primary"
                        : "border-border/60 bg-muted/10 hover:bg-muted/20"
                    }`}
                  >
                    <div className="flex items-center gap-2 font-bold text-foreground">
                      <div className={`size-4 rounded-full border flex items-center justify-center ${formStrategy === "fallback" ? "border-primary bg-primary text-primary-foreground" : "border-muted-foreground"}`}>
                        {formStrategy === "fallback" && <Check className="size-2.5" />}
                      </div>
                      <span>Smart Fallback</span>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      Failover otomatis ke target berikutnya jika provider sebelumnya error atau terkena limit.
                    </p>
                  </div>

                  <div
                    onClick={() => setFormStrategy("round_robin")}
                    className={`cursor-pointer rounded-2xl border p-4 transition ${
                      formStrategy === "round_robin"
                        ? "border-primary bg-primary/5 ring-1 ring-primary"
                        : "border-border/60 bg-muted/10 hover:bg-muted/20"
                    }`}
                  >
                    <div className="flex items-center gap-2 font-bold text-foreground">
                      <div className={`size-4 rounded-full border flex items-center justify-center ${formStrategy === "round_robin" ? "border-primary bg-primary text-primary-foreground" : "border-muted-foreground"}`}>
                        {formStrategy === "round_robin" && <Check className="size-2.5" />}
                      </div>
                      <span>Round Robin (Rolling)</span>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      Memutar urutan prioritas setiap kali ada request baru untuk meratakan distribusi beban.
                    </p>
                  </div>
                </div>
              </div>

              {/* Target Chain Builder */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                    Daftar Target (Urutan Prioritas)
                  </label>
                  <span className="text-xs text-muted-foreground">{formTargets.length} target dipilih</span>
                </div>

                {/* Current Target List */}
                <div className="space-y-2 mb-3">
                  {formTargets.length === 0 ? (
                    <div className="rounded-xl border border-dashed border-border p-4 text-center text-xs text-muted-foreground">
                      Belum ada target. Tambahkan provider dan model di bawah.
                    </div>
                  ) : (
                    formTargets.map((target, idx) => {
                      const info = getTargetBadge(target)
                      return (
                        <div
                          key={idx}
                          className="flex items-center justify-between rounded-xl border border-border/60 bg-card/80 px-3.5 py-2.5 shadow-sm"
                        >
                          <div className="flex items-center gap-3">
                            <span className="flex size-6 items-center justify-center rounded-lg bg-muted text-xs font-black">
                              {idx + 1}
                            </span>
                            <span className={`rounded-lg border px-2 py-0.5 text-xs font-bold ${info.color}`}>
                              {info.provName}
                            </span>
                            <span className="font-mono text-sm font-semibold">{info.modelName}</span>
                          </div>

                          <div className="flex items-center gap-1">
                            <button
                              type="button"
                              onClick={() => handleMoveTarget(idx, "up")}
                              disabled={idx === 0}
                              className="rounded-lg p-1 text-muted-foreground hover:bg-muted disabled:opacity-30"
                              title="Pindah ke atas"
                            >
                              <MoveUp className="size-4" />
                            </button>
                            <button
                              type="button"
                              onClick={() => handleMoveTarget(idx, "down")}
                              disabled={idx === formTargets.length - 1}
                              className="rounded-lg p-1 text-muted-foreground hover:bg-muted disabled:opacity-30"
                              title="Pindah ke bawah"
                            >
                              <MoveDown className="size-4" />
                            </button>
                            <button
                              type="button"
                              onClick={() => handleRemoveTarget(idx)}
                              className="rounded-lg p-1 text-rose-500 hover:bg-rose-500/10"
                              title="Hapus target"
                            >
                              <Trash2 className="size-4" />
                            </button>
                          </div>
                        </div>
                      )
                    })
                  )}
                </div>

                {/* Add Target Selector */}
                <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                  <div className="text-xs font-bold text-muted-foreground uppercase tracking-wider mb-2">
                    Tambah Target ke Rantai
                  </div>
                  <div className="grid gap-3 sm:grid-cols-3">
                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">Provider</label>
                      <select
                        value={selectedProvider}
                        onChange={e => handleProviderChange(e.target.value)}
                        className="w-full rounded-xl border border-border/60 bg-card px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/40"
                      >
                        {PROVIDER_OPTIONS.map(p => (
                          <option key={p.id} value={p.id}>
                            {p.name}
                          </option>
                        ))}
                      </select>
                    </div>

                    <div>
                      <label className="block text-xs font-semibold text-muted-foreground mb-1">Model</label>
                      <select
                        value={selectedModel}
                        onChange={e => setSelectedModel(e.target.value)}
                        className="w-full rounded-xl border border-border/60 bg-card px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/40"
                      >
                        {PROVIDER_OPTIONS.find(p => p.id === selectedProvider)?.models.map(m => (
                          <option key={m.id} value={m.id}>
                            {m.label} ({m.id})
                          </option>
                        ))}
                      </select>
                    </div>

                    <div className="flex items-end">
                      <Button
                        type="button"
                        onClick={handleAddTarget}
                        variant="secondary"
                        className="w-full rounded-xl flex items-center justify-center gap-1.5 font-bold"
                      >
                        <Plus className="size-4" />
                        <span>Tambahkan</span>
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              {/* Actions */}
              <div className="mt-6 flex justify-end gap-3 border-t border-border/50 pt-4">
                <Button variant="outline" onClick={() => setModalOpen(false)} className="rounded-xl">
                  Batal
                </Button>
                <Button onClick={handleSaveCombo} className="rounded-xl bg-primary text-primary-foreground font-bold">
                  {editingCombo ? "Simpan Perubahan" : "Buat Combo Model"}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
