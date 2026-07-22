import './Button.css';

export default function Button({ variant = 'secondary', className = '', children, ...props }) {
  return (
    <button className={`ui-button ui-button-${variant} ${className}`.trim()} {...props}>
      {children}
    </button>
  );
}
