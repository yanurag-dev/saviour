import React from 'react';
import './DataTable.css';

interface Column<T> {
  key: string;
  label: string;
  render?: (item: T) => React.ReactNode;
  align?: 'left' | 'center' | 'right';
}

interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (item: T) => string;
}

export function DataTable<T>({ data, columns, keyExtractor }: DataTableProps<T>) {
  return (
    <div className="data-table-container">
      <table className="data-table">
        <thead>
          <tr>
            {columns.map((col) => (
              <th key={col.key} className={`data-table__th data-table__th--${col.align || 'left'}`}>
                {col.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="data-table__empty">
                No data available
              </td>
            </tr>
          ) : (
            data.map((item) => (
              <tr key={keyExtractor(item)}>
                {columns.map((col) => (
                  <td key={col.key} className={`data-table__td data-table__td--${col.align || 'left'}`}>
                    {col.render ? col.render(item) : String((item as any)[col.key])}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
