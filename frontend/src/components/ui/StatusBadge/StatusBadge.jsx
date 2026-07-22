import './StatusBadge.css';

const DEFAULT_LABEL = {
  pending: 'Đang chờ',
  submitted: 'Đã nộp',
  graded: 'Đã chấm',
  failed: 'Lỗi',
};

export default function StatusBadge({ status, label, className = '' }) {
  return (
    <span className={`ui-status-badge ui-status-badge-${status}${className ? ` ${className}` : ''}`}>
      {label || DEFAULT_LABEL[status] || status}
    </span>
  );
}
