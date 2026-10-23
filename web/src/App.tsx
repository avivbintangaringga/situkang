import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

function App() {
  return (
    <main className="bg-background text-foreground flex min-h-dvh flex-col items-center justify-center gap-8 px-6 py-10">
      <div className="space-y-2 text-center">
        <h1 className="text-3xl font-semibold tracking-tight">SiTukang</h1>
        <p className="text-muted-foreground text-sm">Sistem Informasi Tukang</p>
      </div>

      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Rekap gaji mingguan</CardTitle>
          <CardDescription>
            Catat absensi harian dan rekap gaji tukang dalam satu tempat.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground text-xs">
            Tahap 1: kerangka aplikasi. Layar absensi dan rekap menyusul.
          </p>
        </CardContent>
        <CardFooter>
          <Button
            variant="outline"
            className="w-full"
            render={<a href="/api/health" />}
          >
            Cek status server
          </Button>
        </CardFooter>
      </Card>
    </main>
  )
}

export default App
