# IELTS Test Data Structure — Spec cho Dev

## Cấu trúc chung (áp dụng cho cả Reading & Listening)

```
Test
 └─ passages[] (Reading) / sections[] (Listening)
     └─ question_groups[]        // 1 group = 1 "Questions X-Y" trong đề thật
         ├─ group_order
         ├─ question_type        // xem bảng bên dưới
         ├─ instructions         // text hiển thị đầu nhóm
         ├─ shared_options[]?    // dùng khi cả nhóm share 1 danh sách lựa chọn
         └─ questions[]
             ├─ question_order   // số thứ tự GLOBAL xuyên suốt toàn bài test
             ├─ text
             ├─ answer
             └─ accepted_answers[]?  // các biến thể được chấp nhận
```

**Nguyên tắc bắt buộc:**
- `question_order` phải liên tục 1→40 (Reading) hoặc 1→40 (Listening) xuyên suốt TOÀN BỘ bài test, không reset theo từng passage/section.
- Mỗi phần tử trong `questions[]` phải có ít nhất 1 ô trống/đáp án thật để chấm — không được có "câu hỏi" thuần hiển thị (header, label) lẫn vào mảng này.
- Khi nhiều câu dùng chung 1 danh sách lựa chọn (matching, word bank) → đưa lên `shared_options` ở group, KHÔNG lặp lại trong từng câu.
- `answer` của câu điền từ luôn là **mảng** các cách viết chấp nhận được, không phải 1 string cố định.

---

## 📖 READING — 11 dạng

| # | question_type | Field đặc thù | Ghi chú chấm điểm |
|---|---|---|---|
| 1 | `true-false-not-given` | — | answer ∈ {TRUE, FALSE, NOT GIVEN}, so sánh không phân biệt hoa/thường |
| 2 | `yes-no-not-given` | — | answer ∈ {YES, NO, NOT GIVEN} |
| 3 | `multiple-choice` | `questions[].options[]` ({id, text}) | 1 đáp án đúng |
| 4 | `multiple-choice-multi` | `select_count`, `options[]` | answer là **mảng 2+ key**, so khớp không phân biệt thứ tự |
| 5 | `matching-headings` | `shared_options[]` (danh sách heading), mỗi câu = 1 đoạn văn | answer là key trỏ vào `shared_options` |
| 6 | `matching-information` | `shared_options[]` (các đoạn A-F, load 1 lần) | 1 đoạn có thể là đáp án cho >1 câu (`allow_reuse: true`) |
| 7 | `matching-features` | `shared_options[]` (tên người/địa điểm) | mỗi lựa chọn thường chỉ dùng 1 lần (`allow_reuse: false`) |
| 8 | `matching-sentence-endings` | `options[]` per câu hoặc `shared_options[]` | nối nửa đầu câu — nửa sau |
| 9 | `sentence-completion` | — | điền input tự do, không word bank, giới hạn số từ (`word_limit`) |
| 10 | `summary-completion` | `has_word_bank`, `word_bank[]?`, `summary_text` (chứa `{{gap}}`) | nếu `word_bank` null → input tự do; nếu có → chọn từ box |
| 11 | `table-completion` | `column_headers[]` (metadata, KHÔNG phải question), `questions[]` = từng hàng | chỉ ô có `{{gap}}` mới là câu hỏi thật |
| — | `short-answer` | `word_limit` | giống sentence-completion nhưng dạng câu hỏi Wh- |
| — | `diagram-label-completion` | `diagram_image_url` | answer là từ/cụm từ lấy từ bài đọc |
| — | `flow-chart-completion` | `flow_structure.steps[]` (chứa `{{gap}}`) | tương tự table nhưng dạng chuỗi bước |

---

## 🎧 LISTENING — 10 dạng (thêm phần riêng cho audio)

**Bổ sung ở cấp Test:**
```
test.audio_url            // 1 file DUY NHẤT cho toàn bài
section.section_start_time / section_end_time   // mốc giây trong file chung
question.timestamp_hint   // mốc giây tuyệt đối, chỉ để UI tua gần đúng, KHÔNG dùng để chấm
```

| # | question_type | Field đặc thù | Ghi chú |
|---|---|---|---|
| 1 | `form-completion` | `form_structure.fields[]` | điền form (tên, ngày, sđt...) |
| 2 | `note-completion` | `note_structure.items[]` (chứa `{{gap}}`) | tương tự table nhưng dạng bullet notes |
| 3 | `table-completion` | `table_structure.rows[]` | giống Reading |
| 4 | `flow-chart-completion` | `flow_structure.steps[]` | giống Reading |
| 5 | `summary-completion` | `summary_text`, `word_bank[]?` | giống Reading |
| 6 | `sentence-completion` | — | điền câu đơn |
| 7 | `multiple-choice` | `options[]` | 1 đáp án |
| 8 | `multiple-choice-multi` | `select_count` | chọn 2+ đáp án |
| 9 | `matching` | `shared_options[]` (ý kiến/đặc điểm) | thường dùng ở Part 3 — nối người nói với ý kiến |
| 10 | `map-plan-labelling` | `map_image_url` (**bắt buộc**), `location_key[]` mô tả từng chữ cái/vị trí | KHÔNG được để trống — nếu thiếu ảnh/mô tả, câu hỏi không dùng được (lỗi đã gặp ở bản trước) |

---

## Checklist khi tạo 1 đề test mới (để đảm bảo cover đủ dạng)

- [ ] Reading: có ít nhất 1 group mỗi dạng trong 11 dạng ở trên, trải đều 3 passage
- [ ] Listening: có ít nhất 1 group mỗi dạng trong 10 dạng ở trên, trải đều 4 section — đặc biệt `map-plan-labelling` phải có `map_image_url` thật
- [ ] Tổng `question_order` = đúng 40 cho mỗi kỹ năng, không trùng, không nhảy số
- [ ] Mọi phần tử `shared_options`/`word_bank` chỉ khai báo 1 lần ở cấp group, không copy lặp trong từng câu
- [ ] Mọi `answer` dạng điền từ là mảng (kể cả khi chỉ có 1 cách viết đúng)