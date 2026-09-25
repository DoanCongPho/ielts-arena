export default function WritingBuilder({ content, onChange }) {
  return (
    <div className="tb-field">
      <label className="tb-field">
        <span>Đề bài (prompt)</span>
        <textarea
          className="tb-textarea"
          rows={6}
          value={content.prompt}
          onChange={(e) => onChange({ ...content, prompt: e.target.value })}
        />
      </label>
      <label className="tb-field">
        <span>URL ảnh biểu đồ (image_url, tuỳ chọn — dùng cho Task 1)</span>
        <input
          className="tb-input"
          value={content.image_url}
          onChange={(e) => onChange({ ...content, image_url: e.target.value })}
        />
      </label>
      <label className="tb-field">
        <span>Bài mẫu (sample_answer, tuỳ chọn — để trống thì AI viết bài mẫu khi chấm)</span>
        <textarea
          className="tb-textarea"
          rows={8}
          value={content.sample_answer || ''}
          onChange={(e) => onChange({ ...content, sample_answer: e.target.value })}
        />
      </label>
    </div>
  );
}
