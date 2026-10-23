# বাকির খাতা — API ও ওয়েবসাইট

[বাকির খাতা](https://github.com/sofolitltd/baki-khata) অ্যাপের সার্ভার: Go + Gin + Neon Postgres। `/` পেজে ল্যান্ডিং সাইট (Tailwind) চলে।

- লাইভ: https://baki-khata-api-silk.vercel.app
- অ্যাপ: [sofolitltd/baki-khata](https://github.com/sofolitltd/baki-khata)

## এন্ডপয়েন্ট
| | |
|---|---|
| `GET /` | ল্যান্ডিং পেজ |
| `GET /health` | সার্ভার ও ডাটাবেসের অবস্থা |
| `POST /auth/register`, `/auth/login` | ফোন + ৫ ডিজিটের PIN (bcrypt; ৫ বার ভুলে ১৫ মিনিট লক) |
| `POST /auth/change-pin` | PIN বদল (লগইন লাগে) |
| `POST /sync/push`, `GET /sync/pull` | সিঙ্ক (লগইন লাগে) |

## পরিবেশ
`DATABASE_URL` ও `JWT_SECRET` লাগে (`.env`/`.env.local` বা Vercel env vars)। এগুলো কখনো গিটে দেবেন না। `PORT` ঐচ্ছিক (ডিফল্ট ৮০৮০)।

## চালানো
```bash
make run       # go run . (পোর্টে পুরোনো সার্ভার থাকলে আগে বন্ধ করে)
make landing   # landing/app.css আবার বানায় (Tailwind)
```

## ডিপ্লয়
`main`-এ push করলে Vercel নিজে ডিপ্লয় করে (`Dockerfile.vercel`, রিজিয়ন `sin1`)।
