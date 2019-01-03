/** @format */

import React, {useState} from 'react' // eslint-disable-line no-unused-vars

const TangleText = require('react-tangle-text')
const urlTrie = require('url-trie')
const reduceObject = require('just-reduce-object')

import {mapToEntryList} from './helpers'
import MainDays from './charts/MainDays'
import MainMonths from './charts/MainMonths'

export default function Data({
  domain,
  today,
  usingMonths,
  days,
  months,
  period,
  updateInterval
}) {
  let [state, replaceState] = useState({
    pagesOpen: false,
    referrersOpen: false,
    referrersTrie: null,
    showSnippet: false
  })
  const setState = change => replaceState({...state, ...change})

  var pages = mapToEntryList(
    usingMonths ? months.compendium.p : days.compendium.p
  )
  let pagesMore = pages.length > 12
  if (!state.pagesOpen) {
    pages = pages.slice(0, 12)
  }

  // the trie magic for referrers
  var refs = mapToEntryList(
    usingMonths ? months.compendium.r : days.compendium.r
  )
  var referrersTrie
  if (state.referrersTrie) {
    referrersTrie = state.referrersTrie
  } else {
    referrersTrie = urlTrie(refs.map(({a, c}) => ({url: a, amount: c})), true)
  }

  var referrers = []
  for (let id in referrersTrie) {
    if (id === 'prev') continue

    let data = referrersTrie[id]
    let amountdeep = data.next
      ? reduceObject(data.next, (acc, _, val) => acc + val.amount, 0)
      : 0
    referrers.push({
      addr: id,
      amountdeep,
      amounthere: data.amount - amountdeep,
      href: data.url,
      more: data.next
    })
  }

  referrers.sort(
    (a, b) => b.amountdeep + b.amounthere - (a.amountdeep + a.amounthere)
  )

  let referrersMore = referrers.length > 12
  if (!state.referrersOpen) {
    referrers = referrers.slice(0, 12)
  }
  // ~

  // var individualSessions
  // if (!props.usingMonths) {
  //   individualSessions = sessionGroupsToIndividual(
  //     props.site.sessionsbyreferrer
  //   )
  // }

  // let totalSessions = props.site.referrers
  //   .map(({c: amount}) => amount)
  //   .reduce((a, b) => a + b, 0)

  return (
    <>
      <div className="container">
        <div className="columns">
          <div className="column is-third">
            <div className="card detail-today has-text-left">
              <div className="card-content">
                <h4 className="subtitle is-4">Pageviews today:</h4>
                <h1 className="title is-1">{today.v || 0}</h1>
              </div>
            </div>
          </div>
          <div className="column is-third">
            <div className="card detail-today has-text-centered">
              <div className="card-content">
                <h4 className="subtitle is-4">Bounces today:</h4>
                <h1 className="title is-1">
                  {typeof today.b === 'number' ? today.b : '-'}
                </h1>
              </div>
            </div>
          </div>
          <div className="column is-third">
            <div className="card detail-today has-text-right">
              <div className="card-content">
                <h4 className="subtitle is-4">Sessions today:</h4>
                <h1 className="title is-1">{today.s || 0}</h1>
              </div>
            </div>
          </div>
        </div>
        <div className="card detail-chart-main">
          <div className="card-header">
            <div className="card-header-title">
              Number of sessions and pageviews
              <TangleChangeLastDays
                updateInterval={updateInterval}
                period={period}
              />
            </div>
          </div>
          <div className="card-image">
            <figure className="image">
              {usingMonths ? (
                <MainMonths months={months} />
              ) : (
                <MainDays days={days} />
              )}
            </figure>
          </div>
        </div>
        {/* {(!props.usingMonths && (
          <div className="card detail-chart-individualsessions">
            <div className="card-header">
              <div className="card-header-title">
                showing {individualSessions.length} sessions{' '}
                <TangleChangeMinScore>{props}</TangleChangeMinScore> from a
                total of {totalSessions}{' '}
              <TangleChangeLastDays updateInterval={updateInterval} period={period} />
              </div>
            </div>
            <div className="card-image">
              <figure className="image">
                <SessionsByReferrer
                  site={props.site}
                  individualSessions={individualSessions}
                  handleClick={props.updateSessionsReferrerSelected}
                />
              </figure>
            </div>
            <div
              className="card-content"
              style={{paddingTop: '3px', paddingBottom: '5px'}}
            >
              <div className="content">
                <p>
                  {props.sessionsReferrerFilter !== undefined
                    ? [
                        'seeing sessions from ',
                        <b>{props.sessionsReferrerFilter}</b>,
                        'only, ',
                        <a onClick={props.dontFilterByReferrer}>
                          view from all?
                        </a>
                      ]
                    : props.sessionsReferrerSelected !== undefined
                      ? [
                          'selected ',
                          <b>{props.sessionsReferrerSelected}</b>,
                          ', ',
                          <a onClick={props.filterByReferrer}>
                            see sessions from this referrer only?
                          </a>
                        ]
                      : 'click at a session bar to selected its referrer.'}
                </p>
              </div>
            </div>
          </div>
        )) ||
          null} */}
        <div className="columns">
          <div className="column is-half">
            <div className="card detail-table">
              <div className="card-header">
                <div className="card-header-title">
                  Most viewed pages
                  <TangleChangeLastDays
                    updateInterval={updateInterval}
                    period={period}
                  />
                </div>
              </div>
              <div className="card-content">
                <table className="table">
                  <tbody>
                    {pages.map(({a: addr, c: amount}) => (
                      <tr key={addr}>
                        <td>
                          {addr + ' '}
                          <a
                            target="_blank"
                            href={`http://${domain}${addr.split('?')[0]}`}
                          >
                            <img
                              src={`data:image/svg+xml,<%3Fxml%20version%3D"1.0"%20encoding%3D"UTF-8"%20standalone%3D"no"%3F><svg%20xmlns%3D"http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg"%20width%3D"12"%20height%3D"12"><path%20fill%3D"%23fff"%20stroke%3D"%2306c"%20d%3D"M1.5%204.518h5.982V10.5H1.5z"%2F><path%20d%3D"M5.765%201H11v5.39L9.427%207.937l-1.31-1.31L5.393%209.35l-2.69-2.688%202.81-2.808L4.2%202.544z"%20fill%3D"%2306f"%2F><path%20d%3D"M9.995%202.004l.022%204.885L8.2%205.07%205.32%207.95%204.09%206.723l2.882-2.88-1.85-1.852z"%20fill%3D"%23fff"%2F><%2Fsvg>`}
                            />
                          </a>
                        </td>
                        <td>{amount}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {pagesMore ? (
                  state.pagesOpen ? (
                    <a
                      onClick={() => {
                        setState({pagesOpen: false})
                      }}
                    >
                      see less
                    </a>
                  ) : (
                    <a
                      onClick={() => {
                        setState({pagesOpen: true})
                      }}
                    >
                      see more
                    </a>
                  )
                ) : null}
              </div>
            </div>
          </div>
          <div className="column is-half">
            <div className="card detail-table">
              <div className="card-header">
                <div className="card-header-title">
                  Top referring sites
                  <TangleChangeLastDays
                    updateInterval={updateInterval}
                    period={period}
                  />
                </div>
              </div>
              <div className="card-content">
                <table className="table">
                  {state.referrersTrie && (
                    <thead>
                      <tr>
                        <td colSpan={3} style={{textAlign: 'right'}}>
                          <a
                            onClick={e => {
                              e.preventDefault()
                              let prev = referrersTrie.prev
                              delete referrersTrie.prev
                              setState({referrersTrie: prev})
                            }}
                          >
                            ↥
                          </a>
                        </td>
                      </tr>
                    </thead>
                  )}
                  <tbody>
                    {referrers.map(
                      ({addr, amounthere, amountdeep, href, more}) => (
                        <tr key={addr}>
                          <td>
                            {addr + ' '}
                            {href &&
                              addr !== '<direct>' && (
                                <a target="_blank" href={href.split('?')[0]}>
                                  <img
                                    src={`data:image/svg+xml,<%3Fxml%20version%3D"1.0"%20encoding%3D"UTF-8"%20standalone%3D"no"%3F><svg%20xmlns%3D"http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg"%20width%3D"12"%20height%3D"12"><path%20fill%3D"%23fff"%20stroke%3D"%2306c"%20d%3D"M1.5%204.518h5.982V10.5H1.5z"%2F><path%20d%3D"M5.765%201H11v5.39L9.427%207.937l-1.31-1.31L5.393%209.35l-2.69-2.688%202.81-2.808L4.2%202.544z"%20fill%3D"%2306f"%2F><path%20d%3D"M9.995%202.004l.022%204.885L8.2%205.07%205.32%207.95%204.09%206.723l2.882-2.88-1.85-1.852z"%20fill%3D"%23fff"%2F><%2Fsvg>`}
                                  />
                                </a>
                              )}
                          </td>
                          <td>{amounthere}</td>
                          <td>
                            {more && [
                              <a
                                onClick={e => {
                                  e.preventDefault()
                                  more.prev = state.referrersTrie
                                  setState({referrersTrie: more})
                                }}
                                title={`other ${amountdeep} URL${
                                  amountdeep !== 1 ? 's in paths' : ' in a path'
                                } after this`}
                              >{`↦ ${amountdeep}`}</a>
                            ]}
                          </td>
                        </tr>
                      )
                    )}
                  </tbody>
                </table>
                {referrersMore ? (
                  state.referrersOpen ? (
                    <a
                      onClick={() => {
                        setState({referrersOpen: false})
                      }}
                    >
                      see less
                    </a>
                  ) : (
                    <a
                      onClick={() => {
                        setState({referrersOpen: true})
                      }}
                    >
                      see more
                    </a>
                  )
                ) : (
                  ''
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

function TangleChangeLastDays({period, updateInterval}) {
  return (
    <small>
      in the last
      <TangleText
        value={period.interval}
        onChange={updateInterval}
        pixelDistance={15}
        min={1}
      />
      {' day' + (period.interval === 1 ? '' : 's')}
    </small>
  )
}

// function TangleChangeMinScore(props) {
//   return (
//     <small>
//       with minimum
//       <TangleText
//         value={props.sessionsMinScore}
//         onChange={props.updateMinScore}
//         pixelDistance={40}
//         min={1}
//         max={99}
//       />
//       score
//     </small>
//   )
// }
