import { useEffect, useState } from "react"
import {
  CheckCircle2,
  Play,
  RefreshCw,
} from "lucide-react"
import { useNavigate } from "react-router-dom"
import { Button } from "../components/ui/button"
import { getAuthHeader } from "../lib/auth"
import { API_BASE } from "../lib/api"
import { toast } from "sonner"

type ProviderItem = {
  id: string
  name: string
  status: string
  type: string
  description: string
  models: string[]
}

const PROVIDER_METADATA: Record<
  string,
  {
    gradient: string
    badgeColor: string
    engineDesc: string
    features: string[]
    recommendedModel: string
  }
> = {
  deepseek: {
    gradient: "from-blue-600/10 via-blue-500/5 to-transparent",
    badgeColor: "bg-blue-500/10 text-blue-700 border-blue-500/30 dark:text-blue-400",
    engineDesc: "Native Go HTTP + Node Wasm SHA-3 Proof-of-Work engine",
    features: [
      "Penalaran R1 (DeepSeek-R1)",
      "V3 General Chat",
      "Streaming SSE Respons Cepat",
      "Bypass Captcha/PoW Mandiri",
    ],
    recommendedModel: "deepseek-reasoner",
  },
  chatgpt: {
    gradient: "from-emerald-600/10 via-emerald-500/5 to-transparent",
    badgeColor: "bg-emerald-500/10 text-emerald-700 border-emerald-500/30 dark:text-emerald-400",
    engineDesc: "Sentinel FNV-1a PoW & Turnstile Token Reverse via Node Helper",
    features: [
      "Model GPT-5/4o & GPT-4o Mini",
      "Native Function / Tool Calling",
      "Local File Mentions (@/file)",
      "Pencarian Web Terpadu",
    ],
    recommendedModel: "gpt-5-6",
  },
  mistral: {
    gradient: "from-amber-600/10 via-amber-500/5 to-transparent",
    badgeColor: "bg-amber-500/10 text-amber-700 border-amber-500/30 dark:text-amber-400",
    engineDesc: "Mistral Le Chat Mobile API Reverse + Sesi Handshake Tanpa Akun",
    features: [
      "Mistral Large Reasoning",
      "Mistral Fast Chat & Codestral",
      "Tool Calling XML / JSON Sieve",
      "Pencarian Web Bawaan Mistral",
    ],
    recommendedModel: "mistral-chat",
  },
  qwen: {
    gradient: "from-purple-600/10 via-purple-500/5 to-transparent",
    badgeColor: "bg-purple-500/10 text-purple-700 border-purple-500/30 dark:text-purple-400",
    engineDesc: "Alibaba Cloud Direct Reverse + Anti-WAF Token Injector",
    features: [
      "Qwen 2.5 Max (Flagship)",
      "Qwen 2.5 72B Coding Agent",
      "WanX 2.1 Multimodal (Gambar & Video)",
      "Pool Multi-Akun Rotasi Otomatis",
    ],
    recommendedModel: "qwen-max",
  },
}

export default function ProvidersPage() {
  const navigate = useNavigate()
  const [providers, setProviders] = useState<ProviderItem[]>([])
  const [loading, setLoading] = useState(true)

  const fetchProviders = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/providers`, { headers: getAuthHeader() })
      if (!res.ok) throw new Error("Gagal mengambil data provider")
      const data = await res.json()
      if (Array.isArray(data.providers)) {
        setProviders(data.providers)
      }
    } catch {
      toast.error("Gagal menghubungi server untuk daftar provider")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchProviders()
  }, [])

  return (
    <div className="space-y-6">
      {/* Header */}
      <section className="relative overflow-hidden rounded-[32px] border border-white/75 bg-card/82 p-6 shadow-[var(--shadow-lift)] backdrop-blur-sm">
        <div className="pointer-events-none absolute -right-20 -top-24 size-64 rounded-full bg-accent/40 blur-3xl" />
        <div className="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div className="text-xs font-black uppercase tracking-[0.28em] text-muted-foreground">Hub Provider Terpadu</div>
            <h2 className="mt-2 text-4xl font-black tracking-tight">Provider AI Terintegrasi</h2>
            <p className="mt-2 max-w-3xl text-muted-foreground leading-relaxed">
              ConvertSW-Gateway menyatukan 4 provider AI utama dengan protokol reverse modern, otomatisasi token handshake, dan bypass proteksi tanpa memerlukan API key berbayar.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-3.5 py-1.5 text-xs font-bold text-emerald-700 ring-1 ring-emerald-500/30 dark:text-emerald-400">
              <span className="size-2 rounded-full bg-emerald-500 animate-pulse" />
              4 Provider Aktif
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={fetchProviders}
              className="rounded-2xl flex items-center gap-1.5 text-xs font-bold"
            >
              <RefreshCw className="size-3.5" />
              <span>Segarkan</span>
            </Button>
          </div>
        </div>
      </section>

      {/* Provider Cards */}
      <div className="grid gap-6 md:grid-cols-2">
        {loading ? (
          <div className="col-span-2 flex h-48 items-center justify-center rounded-[30px] border border-white/75 bg-card/86">
            <div className="flex items-center gap-3 text-muted-foreground">
              <RefreshCw className="size-5 animate-spin" />
              <span>Memuat status provider...</span>
            </div>
          </div>
        ) : (
          providers.map(prov => {
            const meta = PROVIDER_METADATA[prov.id] || {
              gradient: "from-stone-500/10 to-transparent",
              badgeColor: "bg-muted text-muted-foreground",
              engineDesc: prov.description,
              features: [],
              recommendedModel: prov.models[0] || "",
            }

            return (
              <div
                key={prov.id}
                className={`relative overflow-hidden rounded-[30px] border border-white/75 bg-gradient-to-br ${meta.gradient} bg-card/90 p-6 shadow-[var(--shadow-lift)] backdrop-blur-sm transition hover:shadow-lg`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="flex items-center gap-2.5">
                      <span className={`rounded-xl border px-3 py-1 text-xs font-black uppercase tracking-wider ${meta.badgeColor}`}>
                        {prov.id}
                      </span>
                      <span className="inline-flex items-center gap-1 text-xs font-bold text-emerald-600">
                        <CheckCircle2 className="size-3.5" />
                        <span>Online</span>
                      </span>
                    </div>
                    <h3 className="mt-2 text-2xl font-black tracking-tight text-foreground">{prov.name}</h3>
                    <p className="mt-1 text-xs text-muted-foreground font-mono">{meta.engineDesc}</p>
                  </div>

                  <Button
                    size="sm"
                    onClick={() => navigate(`/test?model=${meta.recommendedModel}`)}
                    className="rounded-xl flex items-center gap-1.5 text-xs font-bold shrink-0"
                  >
                    <Play className="size-3 text-emerald-400" />
                    <span>Uji Chat</span>
                  </Button>
                </div>

                {/* Features list */}
                <div className="mt-4 space-y-1.5">
                  <div className="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
                    Kemampuan & Fitur:
                  </div>
                  <div className="grid grid-cols-2 gap-1.5">
                    {meta.features.map((feat, idx) => (
                      <div key={idx} className="flex items-center gap-1.5 text-xs text-foreground/80">
                        <CheckCircle2 className="size-3 text-emerald-500 shrink-0" />
                        <span className="truncate">{feat}</span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Models List */}
                <div className="mt-5 border-t border-border/50 pt-4">
                  <div className="text-[11px] font-bold uppercase tracking-wider text-muted-foreground mb-2">
                    Model Tersedia ({prov.models.length}):
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {prov.models.map(m => (
                      <button
                        key={m}
                        onClick={() => navigate(`/test?model=${m}`)}
                        className="rounded-lg border border-border/60 bg-card/80 px-2.5 py-1 font-mono text-xs font-medium text-foreground/85 transition hover:border-primary hover:bg-primary/5"
                        title={`Uji model ${m}`}
                      >
                        {m}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Feature Compatibility Matrix */}
      <section className="overflow-hidden rounded-[30px] border border-white/75 bg-card/86 shadow-[var(--shadow-lift)]">
        <div className="border-b border-border/50 bg-muted/10 px-6 py-5">
          <h3 className="text-xl font-black tracking-tight">Matriks Kompatibilitas Provider</h3>
          <p className="text-sm text-muted-foreground">
            Dukungan fitur canggih (Tool Calling, Mention File, Web Search, Streaming SSE) di ConvertSW-Gateway.
          </p>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full min-w-[700px] text-left text-sm">
            <thead className="border-b bg-muted/25 text-xs uppercase tracking-wider text-muted-foreground">
              <tr>
                <th className="px-6 py-3 font-bold">Provider</th>
                <th className="px-4 py-3 font-bold">Tipe Eksekusi</th>
                <th className="px-4 py-3 text-center font-bold">Tool Calling</th>
                <th className="px-4 py-3 text-center font-bold">File Mention (@/file)</th>
                <th className="px-4 py-3 text-center font-bold">Web Search</th>
                <th className="px-4 py-3 text-center font-bold">Streaming SSE</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/50">
              <tr>
                <td className="px-6 py-4 font-bold">DeepSeek AI</td>
                <td className="px-4 py-4 text-xs font-mono">Go + Wasm PoW</td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-bold">ChatGPT (OpenAI)</td>
                <td className="px-4 py-4 text-xs font-mono">Node Sentinel PoW</td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-bold">Mistral AI</td>
                <td className="px-4 py-4 text-xs font-mono">Node Mobile API Reverse</td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-bold">Qwen AI</td>
                <td className="px-4 py-4 text-xs font-mono">Go Anti-WAF Pool</td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
                <td className="px-4 py-4 text-center"><CheckCircle2 className="size-4 text-emerald-500 mx-auto" /></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
