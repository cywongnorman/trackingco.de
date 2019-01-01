/** @format */

import React from 'react' // eslint-disable-line no-unused-vars
import {
  ResponsiveContainer,
  ComposedChart,
  XAxis,
  YAxis,
  Tooltip,
  Bar,
  Line
} from 'recharts'

const n = require('format-number')({})

import {formatmonth, mergeColours} from '../helpers'

export default function MainDays({colours = {}, months}) {
  colours = mergeColours(this.props.colours)

  let data = months.map(({month, s, v, b}) => ({
    month,
    s,
    v,
    b
  }))
  let dataMax = Math.max(months.map(({v}) => v))

  return (
    <ResponsiveContainer height={300} width="100%">
      <ComposedChart data={data}>
        <XAxis dataKey="month" hide={true} />
        <YAxis scale="linear" domain={[0, dataMax]} orientation="right" />
        <Tooltip isAnimationActive={false} content={CustomTooltip} />
        <Bar dataKey="s" fill={colours.bar1} />
        <Line
          dataKey="v"
          stroke={colours.line1}
          type="monotone"
          strokeWidth={1}
        />
      </ComposedChart>
    </ResponsiveContainer>
  )
}

const CustomTooltip = function(props) {
  return (
    <div className="custom-tooltip">
      <p className="recharts-tooltip-label">{formatmonth(props.label)}</p>
      <ul className="recharts-tooltip-item-list">
        {props.payload.reverse().map(item => (
          <li className="recharts-tooltip-item" style={{color: item.color}}>
            <span className="recharts-tooltip-item-name">
              {names[item.name]}
            </span>
            <span className="recharts-tooltip-item-separator">:</span>
            <span className="recharts-tooltip-item-value">{n(item.value)}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

const names = {
  s: 'unique sessions',
  v: 'all pageviews'
}
