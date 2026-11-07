# হিসাব সাথী — API ও ওয়েবসাইট

[হিসাব সাথী](https://github.com/sofolitltd/hisab-sathi-app) অ্যাপের সার্ভার: Go + Gin + Neon Postgres। `/` পেজে ল্যান্ডিং সাইট (Tailwind) চলে।

- লাইভ: https://bakikhatabd.vercel.app
- অ্যাপ: [sofolitltd/hisab-sathi-app](https://github.com/sofolitltd/hisab-sathi-app)

## এন্ডপয়েন্ট
| | |
|---|---|
| `GET /` | ল্যান্ডিং পেজ |
| `GET /health` | সার্ভার ও ডাটাবেসের অবস্থা |
| `POST /auth/register`, `/auth/login` | ফোন + ৫ ডিজিটের PIN (bcrypt; ৫ বার ভুলে ১৫ মিনিট লক) |
| `POST /auth/change-pin` | PIN বদল (লগইন লাগে) |
| `POST /sync/push`, `GET /sync/pull` | সিঙ্ক (লগইন লাগে) |

## কোড সাজানো (feature-first)
```
main.go                    রুট আর মিডলওয়্যার জোড়া লাগানো
internal/
  database/                Neon কানেকশন + schema.sql (চালু হলেই বসে, idempotent)
  auth/                    রেজিস্টার, লগইন, PIN বদল, JWT ও RequireAuth
  datasync/                /sync/push ও /sync/pull, DTO
  landing/                 ল্যান্ডিং পেজ; web/ এ HTML, CSS, ছবি (বাইনারিতে embed)
```
কোনো ফাইল ৩০০ লাইনের বেশি না।

## পরিবেশ
`DATABASE_URL` ও `JWT_SECRET` লাগে (`.env`/`.env.local` বা Vercel env vars)। এগুলো কখনো গিটে দেবেন না। `PORT` ঐচ্ছিক (ডিফল্ট ৮০৮০)।

## চালানো
```bash
make run       # go run . (পোর্টে পুরোনো সার্ভার থাকলে আগে বন্ধ করে)
make landing   # internal/landing/web/app.css আবার বানায় (Tailwind)
```

## ডিপ্লয়
`main`-এ push করলে Vercel নিজে ডিপ্লয় করে (`Dockerfile.vercel`, রিজিয়ন `sin1`)। নতুন Vercel প্রজেক্টে `DATABASE_URL` ও `JWT_SECRET` env var সেট না থাকলে ফাংশন ক্র্যাশ করে (`FUNCTION_INVOCATION_FAILED`)।
