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

import {formatdate, mergeColours} from '../helpers'
import customTooltipComponent from './customTooltip'

const CustomTooltip = customTooltipComponent(formatdate)

export default function MainDays({colours = {}, days}) {
  colours = mergeColours(colours)

  let data = days.days.map((day, i) => ({
    day: day.day,
    s: day.s,
    v: day.v,
    b: day.b
  }))
  let dataMax = Math.max(days.days.map(({v}) => v))

  return (
    <ResponsiveContainer height={300} width="100%">
      <ComposedChart data={data}>
        <XAxis dataKey="day" hide={true} />
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
