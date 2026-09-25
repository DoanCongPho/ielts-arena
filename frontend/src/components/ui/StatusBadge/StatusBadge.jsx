import './StatusBadge.css';

const DEFAULT_LABEL = {
  pending: 'Đang chờ',
  submitted: 'Đã nộp',
  grading: 'Đang chấm',
  graded: 'Đã chấm',
  failed: 'Lỗi',
};

// busy adds a pulsing dot for work still in progress (a submission being
// graded in the background).
export default function StatusBadge({ status, label, className = '', busy = false }) {
  return (
    <span className={`ui-status-badge ui-status-badge-${status}${className ? ` ${className}` : ''}`}>
      {busy && <span className="ui-status-badge-dot" aria-hidden />}
      {label || DEFAULT_LABEL[status] || status}
    </span>
  );
}
