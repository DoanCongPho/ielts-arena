# IELTS Arena

IELTS Arena is a web app for IELTS practice, starting with Writing, built on a pure-Go (no framework) backend with a separate frontend that talks to it over REST and WebSocket. Learners pick a Task 1 or Task 2 prompt, write within a timer, and get automated band scores across the four IELTS criteria plus corrections and improvement suggestions; a background worker keeps the question bank current by fetching and normalizing tests from configured sources. On top of solo practice, a real-time Dual Match mode pits multiple users against the same prompt where the highest overall band wins, with a future 4v4 team mode that assigns each player a skill role and combines their results into a team overall. Progress is gamified through levels (from practice XP) and competitive rank (ELO-style), unlocking avatar frames, badges, and titles so users feel their growth is rewarded.

## Chạy dự án

```bash
cp .env.example .env     # điền DB_USER / DB_PASSWORD / DB_NAME / SECRET_KEY / OPENAI_API_KEY
docker compose up --build
```

Mở http://localhost:8081 — đó là cổng duy nhất cần biết. nginx phục vụ SPA và
proxy `/api` + `/assets` sang API, nên mọi request đều same-origin và không có
CORS ở bất kỳ đâu trong stack.

Chạy tách rời khi phát triển:

```bash
make migrate-up && make run     # API ở :8080
cd frontend && npm install && npm run dev
```

Dev server của Vite proxy `/api` sang `localhost:8080`, giống nginx làm ở
production, nên code ứng dụng không bao giờ biết host của API.

## Kiến trúc

- **Backend** — Go thuần (chỉ `gorilla/mux` để routing), `database/sql` không ORM.
  `internal/platform` là hạ tầng dùng chung, `internal/feature` là nghiệp vụ.
  Trong mỗi feature, `Handler` giữ phần HTTP và `Service` giữ phần nghiệp vụ —
  `Service` không tham chiếu tới router, request hay status code nào.
- **Chấm bài chạy nền** — `POST /api/submissions` trả `202` kèm một submission
  ở trạng thái `pending`; một pool worker (`internal/feature/ielts_test/worker.go`)
  nhận việc từ hàng đợi, chấm, rồi chuyển sang `graded`/`failed`. Client poll
  `GET /api/submissions/{id}`. Lỗi tạm thời được thử lại với backoff tăng dần;
  worker chết giữa chừng thì lease hết hạn và worker khác nhận lại.
- **Frontend** — React + Vite, không có state manager: `lib/api.js` là tầng gọi
  API duy nhất, các trang tự fetch.

## Trạng thái

**Đã có:** đăng ký/đăng nhập JWT (access + refresh), luyện tập 4 kỹ năng với
Reading/Listening chấm tự động qua 18 loại câu hỏi, Writing chấm bằng LLM theo
4 tiêu chí IELTS kèm sửa lỗi và bài mẫu, che đáp án phía server, lịch sử làm
bài, hệ thống level/XP với 100 khung avatar, trang tạo đề cho admin.

**Dự kiến:** Dual Match thời gian thực qua WebSocket, chế độ 4v4 theo đội,
xếp hạng kiểu ELO, huy hiệu và danh hiệu, worker tự thu thập đề từ nguồn ngoài.

