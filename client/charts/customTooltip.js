/** @format */

import React from 'react' // eslint-disable-line no-unused-vars

const n = require('format-number')({})

export default function customTooltipComponent(formatfunction) {
  return function CustomTooltip(props) {
    if (!props.payload) return <div />

    return (
      <div className="custom-tooltip">
        <p className="recharts-tooltip-label">{formatfunction(props.label)}</p>
        <ul className="recharts-tooltip-item-list">
          {props.payload.reverse().map(item => (
            <li
              key={item.value}
              className="recharts-tooltip-item"
              style={{color: item.color}}
            >
              <span className="recharts-tooltip-item-name">
                {names[item.name]}
              </span>
              <span className="recharts-tooltip-item-separator">: </span>
              <span className="recharts-tooltip-item-value">
                {n(item.value)}
              </span>
            </li>
          ))}
        </ul>
      </div>
    )
  }
}

const names = {
  s: 'unique sessions',
  v: 'all pageviews',
  b: 'bounce sessions',
  c: 'total score'
}
