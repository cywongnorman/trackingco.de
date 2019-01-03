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

import {formatmonth, mergeColours} from '../helpers'
import customTooltipComponent from './customTooltip'

const CustomTooltip = customTooltipComponent(formatmonth)

export default function MainDays({colours = {}, months}) {
  colours = mergeColours(colours)

  let data = months.months.map(({month, s, v, b}) => ({
    month,
    s,
    v,
    b
  }))
  let dataMax = Math.max(months.months.map(({v}) => v))

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
