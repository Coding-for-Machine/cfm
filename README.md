# 🌐 CFM

CFM — bu [cofm](https://github.com/...) loyihasidan ilhomlangan, lekin **Go fasthttp** kutubxonasi asosida yozilgan tezkor va yengil HTTP tunneling xizmati.  
U orqali sen localhost’da ishlayotgan ilovangni tashqi tarmoqdan ham xavfsiz va tezkor tarzda ishlatishing mumkin.

---

## ✨ Xususiyatlari
- 🚀 **Yuqori tezlik** — fasthttp yordamida minimal latency va yuqori throughput.  
- 🔒 **Xavfsiz ulanish** — WebSocket asosida tunellash.  
- 🌍 **Public URL yaratish** — lokal portni tashqi dunyoga ochish.  
- 📊 **Dashboard** — ulanishlar monitoringi.  
- ⚡ **Yengil va modul** — oddiy sozlash, container-friendly.

---

## 📦 O‘rnatish

```bash
# Serverni build qilish
go build -o bin/server cmd/server/main.go

# Clientni build qilish
go build -o bin/client cmd/client/main.go
```