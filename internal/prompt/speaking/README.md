# Prompt sửa bài speaking theo từng part

Mỗi file `partN.md` (N = 1, 2, 3) là prompt sửa bài và viết bài mẫu cho Part N.

- **Có file** thì mỗi bài có Part đó được sửa bằng prompt này. Model chỉ nhận câu trả lời của **part đó**:
  - Part 2 nhận cue card và phần nói.
  - Part 1 và Part 3 nhận các câu hỏi cùng câu trả lời.
- **Không có file** thì Part đó không được sửa và không tốn token.
- Các part chạy song song với phần chấm band, không làm chấm lâu hơn. Nếu một part lỗi, bài vẫn được chấm, chỉ thiếu phần sửa của part đó.

Đầu file có thể khai báo cách hiển thị trên trang kết quả (không bắt buộc):

```
---
title: Bài mẫu Part 1
badge: Đặc biệt dành cho học sinh thầy Sơn
note: Một câu giới thiệu ngắn hiện dưới tiêu đề.
---
# Nội dung prompt...
```

Phần còn lại của file được gửi nguyên văn làm system prompt. App tự thêm yêu cầu trả về JSON ở cuối nên prompt không cần nói tới định dạng JSON. Sửa hoặc thêm file xong thì build và deploy lại (file được nhúng vào backend lúc build).
