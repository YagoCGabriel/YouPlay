# YouPlay v2.0 — Go

CLI para buscar e assistir vídeos do YouTube via yt-dlp + player externo.

---

## Dependências

- [Go 1.22+](https://go.dev/dl/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) no PATH
- Um player: **[mpv](https://mpv.io/)** (recomendado), vlc, potplayer etc.

---

## Build

### Padrão
```bash
go mod tidy
go build -o youplay.exe .
```

### Otimizado (sem símbolos de debug, ~30% menor)
```bash
go build -ldflags="-s -w" -o youplay.exe .
```

### Cross-compile (Linux/macOS → Windows)
```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o youplay.exe .
```

### Compactar com UPX (~50-60% menor)
```bash
go build -ldflags="-s -w" -o youplay.exe .
upx --best youplay.exe
```

> O `.exe` gerado é **standalone** — não precisa instalar Go na máquina de destino.
> Apenas `yt-dlp.exe` e o player precisam estar no PATH.

### Rodar sem compilar
```bash
go run .
```

---

## Estrutura

```
youplay/
  main.go       # código completo
  go.mod        # dependências
  LICENSE       # MIT
  data/
    config.json # criado automaticamente na primeira execução
```

---

## Players suportados

| Player       | yt-dlp nativo | Merge streams        |
|--------------|:-------------:|:--------------------:|
| mpv          | ✅            | ✅                   |
| vlc          | ❌            | ⚠ fallback mp4      |
| potplayer    | ❌            | ⚠ fallback mp4      |
| potplayer64  | ❌            | ⚠ fallback mp4      |
| kmplayer     | ❌            | ⚠ fallback mp4      |
| outro        | ❌            | resolve e abre       |

---

## Qualidades disponíveis

| Opção               | Formato                                    |
|---------------------|--------------------------------------------|
| 🏆 Melhor disponível | bestvideo+bestaudio                        |
| 🖥  1080p            | h264 mp4 + m4a ≤128kbps                   |
| 📺 720p             | h264 mp4 + m4a ≤128kbps                   |
| 📱 480p             | bestvideo+bestaudio ≤96kbps até 480p       |
| 🎵 Apenas áudio     | bestaudio                                  |

> 1080p e 720p limitam o áudio a **128kbps m4a** para liberar banda pro vídeo
> e evitar gargalos causados pela competição entre os dois streams DASH paralelos.
> A diferença de qualidade de áudio é inaudível em streaming.

---

## Flags MPV aplicadas automaticamente

| Flag | Efeito |
|------|--------|
| `--hwdec=auto-safe` | Decodificação por hardware estável |
| `--vo=gpu --gpu-api=d3d11` | Renderização via Direct3D 11 |
| `--profile=gpu-hq` | Perfil de alta qualidade |
| `--scale=lanczos` | Upscaling de melhor qualidade |
| `--video-sync=display-resample` | Evita tearing |
| `--cache=yes --cache-secs=120` | Pré-carrega 2 minutos à frente |
| `--demuxer-max-bytes=150MiB` | Buffer de demux generoso |
| `--demuxer-readahead-secs=20` | Leitura antecipada de 20s |
| `--network-timeout=30` | Timeout de rede |
| `--audio-buffer=2.0` | Buffer de áudio de 2s |
| `--demuxer-lavf-analyzeduration=10` | Mais tempo para sincronizar os streams |
| `concurrent-fragments=2` | 2 fragmentos paralelos — evita competição de banda |
| `retries=10,fragment-retries=10` | Recuperação de falhas de rede |

---

## Licença

MIT © 2026 YouPlay
