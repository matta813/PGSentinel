import type { ReactNode } from "react";

export function Table({
  headers,
  rows,
  numeric = [],
}: {
  headers: ReactNode[];
  rows: ReactNode[][];
  numeric?: number[];
}) {
  return (
    <div className="data-table">
      <table>
        <thead>
          <tr>
            {headers.map((header, index) => (
              <th
                className={numeric.includes(index) ? "numeric" : ""}
                key={index}
              >
                {header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, rowIndex) => (
            <tr key={rowIndex}>
              {row.map((value, columnIndex) => (
                <td
                  className={numeric.includes(columnIndex) ? "numeric" : ""}
                  key={columnIndex}
                >
                  {value}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
