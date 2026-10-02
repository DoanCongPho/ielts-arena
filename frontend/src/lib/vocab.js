// Turning a review's highlights into a vocabulary list, and the prompt a
// learner pastes into ChatGPT, Claude or Gemini to get it back as a CSV
// for luyentu.com (columns as its import screen names them).

export const VOCAB_COLUMNS = ['Từ vựng', 'Phiên âm', 'Nghĩa', 'Loại từ', 'Ví dụ', 'Ghi chú'];

// cleanWord trims the punctuation a selection tends to catch at its edges.
function cleanWord(s) {
  return s
    .replace(/\s+/g, ' ')
    .replace(/^[\s"'“‘([–—-]+|[\s"'”’)\],.;:!?–—-]+$/g, '')
    .trim();
}

// wholeWords widens [start, end) to whole words, as a selection often
// stops a letter or two short.
function wholeWords(text, start, end) {
  const inWord = (c) => /[\p{L}\p{N}'’-]/u.test(c || '');
  while (start > 0 && inWord(text[start - 1]) && inWord(text[start])) start--;
  while (end < text.length && inWord(text[end]) && inWord(text[end - 1])) end++;
  return [start, end];
}

// sentenceAround is the sentence of `text` that holds [start, end), to
// give the word its meaning in context.
function sentenceAround(text, start, end) {
  const before = text.slice(0, start);
  const from = Math.max(before.lastIndexOf('. '), before.lastIndexOf('? '), before.lastIndexOf('! '));
  const after = text.slice(end).search(/[.?!](\s|$)/);
  const sentence = text.slice(from === -1 ? 0 : from + 2, after === -1 ? text.length : end + after + 1).trim();
  return sentence.length > 240 ? `${sentence.slice(0, 237)}…` : sentence;
}

// collectWords lists the highlighted words across every passage/section
// (`highlights[scope][key]` ranges into `texts[scope][key]`), each once
// whatever its case, with the sentence of its first occurrence and where
// every occurrence is (`at`), so removing it clears them all.
export function collectWords(highlights, texts) {
  const seen = new Map();
  const words = [];
  for (const [scope, byKey] of Object.entries(highlights || {})) {
    for (const [key, ranges] of Object.entries(byKey || {})) {
      const text = texts?.[scope]?.[key];
      if (!text) continue;
      for (const { start, end } of ranges) {
        const word = cleanWord(text.slice(...wholeWords(text, start, end)));
        const id = word.toLowerCase();
        if (!word) continue;
        const at = { scope, key, start, end };
        if (seen.has(id)) {
          seen.get(id).at.push(at);
          continue;
        }
        const entry = { word, sentence: sentenceAround(text, start, end), at: [at] };
        seen.set(id, entry);
        words.push(entry);
      }
    }
  }
  return words;
}

// testSource names where the words come from, for the prompt: "Cambridge
// 16 Reading Test 2", or just the skill outside a series.
export function testSource(test, skillLabel) {
  if (!test?.series) return `bài ${skillLabel}`;
  const book = test.series === 'cambridge' ? 'Cambridge' : test.series;
  return [`${book}${test.volume ? ` ${test.volume}` : ''}`, skillLabel, test.test_number ? `Test ${test.test_number}` : ''].filter(Boolean).join(' ');
}

// buildVocabPrompt is the instruction for an LLM to fill in the columns
// for each word, with the sentence it came from so the meaning fits.
export function buildVocabPrompt(words, source) {
  const list = words.map((w, i) => `${i + 1}. ${w.word} — câu gốc: "${w.sentence}"`).join('\n');
  return `Bạn là trợ lý học từ vựng IELTS cho người Việt. Hãy tạo một file CSV cho các từ/cụm từ bên dưới${source ? ` (lấy từ ${source})` : ''}.

Dòng đầu là tiêu đề, đúng các cột và thứ tự sau:
${VOCAB_COLUMNS.join(',')}

Quy tắc cho từng cột:
- Từ vựng: dạng gốc trong từ điển (động từ nguyên thể, danh từ số ít); giữ nguyên cụm từ nếu là cụm.
- Phiên âm: IPA giọng Anh-Anh, đặt trong /.../.
- Nghĩa: nghĩa tiếng Việt ngắn gọn, đúng với nghĩa trong câu gốc.
- Loại từ: noun, verb, adjective, adverb hoặc phrase.
- Ví dụ: một câu tiếng Anh mới, ngắn và tự nhiên (không chép lại câu gốc).
- Ghi chú: 1-2 từ đồng nghĩa hoặc collocation hay đi kèm.

Mỗi từ một dòng, mọi ô đặt trong dấu ngoặc kép, mã hoá UTF-8.

Trả về kết quả dưới dạng một FILE tải về được tên tu-vung.csv (dùng công cụ tạo file hoặc chạy code của bạn), không dán nội dung CSV vào câu trả lời. Chỉ khi bạn không tạo được file mới trả về nội dung CSV trong một khối code. Không cần giải thích gì thêm.

Danh sách từ:
${list}`;
}
