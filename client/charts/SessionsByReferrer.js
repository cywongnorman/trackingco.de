/** @format */

import React from 'react' // eslint-disable-line no-unused-vars

const R = require('recharts')

const referrerColour = require('../helpers').referrerColour

export default class SessionsByReferrer extends React.Component {
  shouldComponentUpdate(nextProps, nextState) {
    if (nextProps.site !== this.props.site) {
      return true
    }
    return false
  }

  render() {
    return (
      <R.ResponsiveContainer height={200} width="100%">
        <R.BarChart data={this.props.individualSessions} barGap="3%">
          <R.XAxis dataKey="referrer" hide={true} />
          <R.Bar dataKey="score" onClick={this.props.handleClick}>
            {this.props.individualSessions.map((session, i) => (
              <R.Cell key={i} fill={referrerColour(session.referrer)} />
            ))}
          </R.Bar>
          <R.Tooltip isAnimationActive={false} content={Tooltip} />
        </R.BarChart>
      </R.ResponsiveContainer>
    )
  }
}

const Tooltip = function(props) {
  return props.payload.length ? (
    <div className="custom-tooltip">
      <ul className="recharts-tooltip-item-list">
        <li
          className="recharts-tooltip-props payload"
          style={{color: props.payload[0].color}}
        >
          <span className="recharts-tooltip-props payload">{props.label}</span>
          <span className="recharts-tooltip-props payload">:</span>
          <span className="recharts-tooltip-props payload">
            {props.payload[0].value}
          </span>
        </li>
      </ul>
    </div>
  ) : (
    <div />
  )
}

export function sessionGroupsToIndividual(sessiongroups) {
  return sessiongroups
    .map(group =>
      group.scores.map(score => ({
        referrer: group.referrer,
        score
      }))
    )
    .reduce((a, b) => a.concat(b), [])
}
