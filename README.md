# YouPlay v2.0 — Go

CLI para buscar e assistir vídeos do YouTube via yt-dlp + player externo.

---

## Dependências

- [Go 1.22+](https://go.dev/dl/)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) no PATH
- Um player: **mpv** (recomendado), vlc, potplayer etc.

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

| Opção               | Formato                          |
|---------------------|----------------------------------|
| 🏆 Melhor disponível | bestvideo+bestaudio              |
| 🖥  1080p            | h264 mp4 preferencial + m4a      |
| 📺 720p             | h264 mp4 preferencial + m4a      |
| 📱 480p             | bestvideo+bestaudio até 480p     |
| 🎵 Apenas áudio     | bestaudio                        |

> 1080p e 720p priorizam **H.264 + m4a** para evitar artefatos com VP9/AV1
> e reduzir gargalos por merge de streams paralelos.

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
| `concurrent-fragments=4` | 4 fragmentos paralelos via yt-dlp |
| `retries=10,fragment-retries=10` | Recuperação de falhas de rede |
