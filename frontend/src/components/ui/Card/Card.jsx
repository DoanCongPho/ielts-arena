import './Card.css';

export default function Card({
  as: Tag = 'div',
  padding = 'default',
  interactive = false,
  className = '',
  children,
  ...props
}) {
  return (
    <Tag
      className={
        `ui-card ui-card-padding-${padding}` +
        (interactive ? ' ui-card-interactive' : '') +
        (className ? ` ${className}` : '')
      }
      {...props}
    >
      {children}
    </Tag>
  );
}
