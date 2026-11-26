import { useEffect, useRef } from 'react';
import Button from '../Button/Button';
import './ConfirmDialog.css';

// ConfirmDialog is an in-app replacement for window.confirm, built on the
// native <dialog> element: showModal() gives it the backdrop, focus trap
// and Escape (which counts as cancel). The safe choice (`cancelLabel`) has
// focus, so Enter never confirms something destructive by accident.
export default function ConfirmDialog({
  open,
  title,
  children,
  confirmLabel = 'Đồng ý',
  cancelLabel = 'Huỷ',
  danger = false,
  busy = false,
  onConfirm,
  onCancel,
}) {
  const ref = useRef(null);
  const cancelRef = useRef(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) {
      dialog.showModal();
      cancelRef.current?.focus();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open]);

  return (
    <dialog
      ref={ref}
      className="ui-confirm-dialog"
      onCancel={(e) => {
        e.preventDefault();
        onCancel?.();
      }}
      onClick={(e) => {
        // A click on the backdrop lands on the <dialog> itself.
        if (e.target === ref.current) onCancel?.();
      }}
    >
      <div className="ui-confirm-dialog-body">
        {title && <h2 className="ui-confirm-dialog-title text-h3">{title}</h2>}
        {children && <div className="ui-confirm-dialog-text text-body">{children}</div>}
        <div className="ui-confirm-dialog-actions">
          <Button ref={cancelRef} variant="primary" onClick={onCancel} disabled={busy}>
            {cancelLabel}
          </Button>
          <Button variant={danger ? 'danger' : 'secondary'} onClick={onConfirm} disabled={busy}>
            {confirmLabel}
          </Button>
        </div>
      </div>
    </dialog>
  );
}
