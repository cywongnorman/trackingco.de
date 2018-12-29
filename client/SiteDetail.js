/** @format */

import React, {useState, useEffect, useRef} from 'react'

const fetch = window.fetch
const TangleText = require('react-tangle-text')
const throttle = require('throttleit')
const DocumentTitle = require('react-document-title')
const urlTrie = require('url-trie')
const reduceObject = require('just-reduce-object')

import log from './log'
import {title} from './helpers'

const charts = {
  MainDays: require('./charts/MainDays'),
  MainMonths: require('./charts/MainMonths'),
  SessionsByReferrer: require('./charts/SessionsByReferrer')
}

export default function SiteDetail(props) {
  let [state, replaceState] = useState({
    site: null,
    dataMax: 100,
    nlastdays: 45,
    usingMonths: false,
    sessionsLimit: 400,
    sessionsMinScore: 1,
    sessionsReferrerSelected: undefined,
    sessionsReferrerFilter: undefined
  })
  const setState = change => replaceState({...state, ...change})

  let rootElement = useRef(null)

  useEffect(() => {
    setSessionsLimit()
    window.addEventListener('resize', setSessionsLimit)

    return () => {
      window.removeEventListener('resize', setSessionsLimit)
    }
  }, [])

  const setSessionsLimit = throttle(function(e) {
    let sessionsLimit = parseInt(rootElement.offsetWidth / 5)
    if (sessionsLimit === state.sessionsLimit) return

    setState({sessionsLimit: sessionsLimit}, query)
  }, 1000)

  async function query(last) {
    last = last || state.nlastdays
    let usingMonths = last > 90

    let path = usingMonths ? '/query/months' : '/query/days'
    let params = {
      domain: props.domain,
      last: last,
      limit: state.sessionsLimit,
      min_score: state.sessionsMinScore,
      referrer_filter: state.sessionsReferrerFilter
    }

    try {
      let res = await fetch(path, {
        method: 'GET',
        body: JSON.stringify(params),
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json'
        }
      })

      if (!res.ok) throw new Error(await res.text())

      let r = await res.json()

      setState({
        nlastdays: last,
        usingMonths,
        dataMax: Math.max(
          ...(r.days || []).map(d => d.v),
          ...(r.months || []).map(d => d.v)
        )
      })
    } catch (e) {
      log.error(e)
    }
  }

  function filterByReferrer() {
    setState({
      sessionsReferrerSelected: undefined,
      sessionsReferrerFilter: state.sessionsReferrerSelected
    })
    query()
  }

  function dontFilterByReferrer() {
    setState({
      sessionsReferrerSelected: undefined,
      sessionsReferrerFilter: undefined
    })
    query()
  }

  if (!state.site) {
    return <div />
  }

  return (
    <div className="container">
      <div className="content">
        <h4 className="title is-3">{state.site.domain}</h4>
      </div>
      {state.dataMax === 0 && state.site.today.v === 0 ? (
        <p>No data.</p>
      ) : (
        <Data
          {...state}
          updateNLastDays={query}
          updateMinScore={v => {
            setState({sessionsMinScore: v}, query)
          }}
          updateSessionsReferrerSelected={data => {
            setState({sessionsReferrerSelected: data.payload.referrer})
          }}
          filterByReferrer={filterByReferrer}
          dontFilterByReferrer={dontFilterByReferrer}
        />
      )}
    </div>
  )
}

class Data extends React.Component {
  constructor(props) {
    super(props)

    this.state = {
      pagesOpen: false,
      referrersOpen: false,
      referrersTrie: null,
      showSnippet: false
    }
  }

  render() {
    let pagesMore = this.props.site.pages.length > 12
    var pages = this.props.site.pages
    if (!this.state.pagesOpen) {
      pages = this.props.site.pages.slice(0, 12)
    }

    // the trie magic for referrers
    var referrersTrie
    if (this.state.referrersTrie) {
      referrersTrie = this.state.referrersTrie
    } else {
      referrersTrie = urlTrie(
        this.props.site.referrers.map(({a, c}) => ({url: a, count: c})),
        true
      )
    }

    var referrers = []
    for (let id in referrersTrie) {
      if (id === 'prev') continue

      let data = referrersTrie[id]
      let countdeep = data.next
        ? reduceObject(data.next, (acc, _, val) => acc + val.count, 0)
        : 0
      referrers.push({
        addr: id,
        countdeep,
        counthere: data.count - countdeep,
        href: data.url,
        more: data.next
      })
    }

    referrers.sort(
      (a, b) => b.countdeep + b.counthere - (a.countdeep + a.counthere)
    )

    let referrersMore = referrers.length > 12
    if (!this.state.referrersOpen) {
      referrers = referrers.slice(0, 12)
    }
    // ~

    var individualSessions
    if (!this.props.usingMonths) {
      individualSessions = charts.SessionsByReferrer.sessionGroupsToIndividual(
        this.props.site.sessionsbyreferrer
      )
    }

    let totalSessions = this.props.site.referrers
      .map(({c: count}) => count)
      .reduce((a, b) => a + b, 0)

    return (
      <DocumentTitle title={title(this.props.site.domain)}>
        <div className="container">
          <div className="columns">
            <div className="column is-third">
              <div className="card detail-today has-text-left">
                <div className="card-content">
                  <h4 className="subtitle is-4">Pageviews today:</h4>
                  <h1 className="title is-1">{this.props.site.today.v || 0}</h1>
                </div>
              </div>
            </div>
            <div className="column is-third">
              <div className="card detail-today has-text-centered">
                <div className="card-content">
                  <h4 className="subtitle is-4">Bounce rate today:</h4>
                  <h1 className="title is-1">
                    {typeof this.props.site.today.b === 'number'
                      ? (this.props.site.today.b * 100).toFixed(1) + '%'
                      : '-'}
                  </h1>
                </div>
              </div>
            </div>
            <div className="column is-third">
              <div className="card detail-today has-text-right">
                <div className="card-content">
                  <h4 className="subtitle is-4">Sessions today:</h4>
                  <h1 className="title is-1">{this.props.site.today.s || 0}</h1>
                </div>
              </div>
            </div>
          </div>
          <div className="card detail-chart-main">
            <div className="card-header">
              <div className="card-header-title">
                Number of sessions and pageviews
                <TangleChangeLastDays>{this.props}</TangleChangeLastDays>
              </div>
            </div>
            <div className="card-image">
              <figure className="image">
                {this.props.usingMonths ? (
                  <charts.MainMonths
                    site={this.props.site}
                    dataMax={this.props.dataMax}
                    colours={this.props.me.colours}
                  />
                ) : (
                  <charts.MainDays
                    site={this.props.site}
                    dataMax={this.props.dataMax}
                    colours={this.props.me.colours}
                  />
                )}
              </figure>
            </div>
          </div>
          {(!this.props.usingMonths && (
            <div className="card detail-chart-individualsessions">
              <div className="card-header">
                <div className="card-header-title">
                  showing {individualSessions.length} sessions{' '}
                  <TangleChangeMinScore>{this.props}</TangleChangeMinScore> from
                  a total of {totalSessions}{' '}
                  <TangleChangeLastDays>{this.props}</TangleChangeLastDays>
                </div>
              </div>
              <div className="card-image">
                <figure className="image">
                  <charts.SessionsByReferrer
                    site={this.props.site}
                    individualSessions={individualSessions}
                    handleClick={this.props.updateSessionsReferrerSelected}
                  />
                </figure>
              </div>
              <div
                className="card-content"
                style={{paddingTop: '3px', paddingBottom: '5px'}}
              >
                <div className="content">
                  <p>
                    {this.props.sessionsReferrerFilter !== undefined
                      ? [
                          'seeing sessions from ',
                          <b>{this.props.sessionsReferrerFilter}</b>,
                          'only, ',
                          <a onClick={this.props.dontFilterByReferrer}>
                            view from all?
                          </a>
                        ]
                      : this.props.sessionsReferrerSelected !== undefined
                        ? [
                            'selected ',
                            <b>{this.props.sessionsReferrerSelected}</b>,
                            ', ',
                            <a onClick={this.props.filterByReferrer}>
                              see sessions from this referrer only?
                            </a>
                          ]
                        : 'click at a session bar to selected its referrer.'}
                  </p>
                </div>
              </div>
            </div>
          )) ||
            null}
          <div className="columns">
            <div className="column is-half">
              <div className="card detail-table">
                <div className="card-header">
                  <div className="card-header-title">
                    Most viewed pages
                    <TangleChangeLastDays>{this.props}</TangleChangeLastDays>
                  </div>
                </div>
                <div className="card-content">
                  <table className="table">
                    <tbody>
                      {pages.map(({a: addr, c: count}) => (
                        <tr>
                          <td>{addr}</td>
                          <td>{count}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {pagesMore ? (
                    this.state.pagesOpen ? (
                      <a
                        onClick={() => {
                          this.setState({pagesOpen: false})
                        }}
                      >
                        see less
                      </a>
                    ) : (
                      <a
                        onClick={() => {
                          this.setState({pagesOpen: true})
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
            <div className="column is-half">
              <div className="card detail-table">
                <div className="card-header">
                  <div className="card-header-title">
                    Top referring sites
                    <TangleChangeLastDays>{this.props}</TangleChangeLastDays>
                  </div>
                </div>
                <div className="card-content">
                  <table className="table">
                    {this.state.referrersTrie && (
                      <thead>
                        <tr>
                          <td colSpan={3} style={{textAlign: 'right'}}>
                            <a
                              onClick={e => {
                                e.preventDefault()
                                let prev = referrersTrie.prev
                                delete referrersTrie.prev
                                this.setState({referrersTrie: prev})
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
                        ({addr, counthere, countdeep, href, more}) => (
                          <tr>
                            <td>
                              {addr + ' '}
                              {href &&
                                addr !== '<direct>' && (
                                  <a target="_blank" href={href}>
                                    <img
                                      src={`data:image/svg+xml,<%3Fxml%20version%3D"1.0"%20encoding%3D"UTF-8"%20standalone%3D"no"%3F><svg%20xmlns%3D"http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg"%20width%3D"12"%20height%3D"12"><path%20fill%3D"%23fff"%20stroke%3D"%2306c"%20d%3D"M1.5%204.518h5.982V10.5H1.5z"%2F><path%20d%3D"M5.765%201H11v5.39L9.427%207.937l-1.31-1.31L5.393%209.35l-2.69-2.688%202.81-2.808L4.2%202.544z"%20fill%3D"%2306f"%2F><path%20d%3D"M9.995%202.004l.022%204.885L8.2%205.07%205.32%207.95%204.09%206.723l2.882-2.88-1.85-1.852z"%20fill%3D"%23fff"%2F><%2Fsvg>`}
                                    />
                                  </a>
                                )}
                            </td>
                            <td>{counthere}</td>
                            <td>
                              {more && [
                                <a
                                  onClick={e => {
                                    e.preventDefault()
                                    more.prev = this.state.referrersTrie
                                    this.setState({referrersTrie: more})
                                  }}
                                  title={`other ${countdeep} URL${
                                    countdeep !== 1
                                      ? 's in paths'
                                      : ' in a path'
                                  } after this`}
                                >{`↦ ${countdeep}`}</a>
                              ]}
                            </td>
                          </tr>
                        )
                      )}
                    </tbody>
                  </table>
                  {referrersMore ? (
                    this.state.referrersOpen ? (
                      <a
                        onClick={() => {
                          this.setState({referrersOpen: false})
                        }}
                      >
                        see less
                      </a>
                    ) : (
                      <a
                        onClick={() => {
                          this.setState({referrersOpen: true})
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
      </DocumentTitle>
    )
  }
}

const TangleChangeLastDays = function(props) {
  return (
    <small>
      in the last
      <TangleText
        value={props.nlastdays}
        onChange={props.updateNLastDays}
        pixelDistance={15}
        min={1}
      />
      {' day' + (props.nlastdays === 1 ? '' : 's')}
    </small>
  )
}

const TangleChangeMinScore = function(props) {
  return (
    <small>
      with minimum
      <TangleText
        value={props.sessionsMinScore}
        onChange={props.updateMinScore}
        pixelDistance={40}
        min={1}
        max={99}
      />
      score
    </small>
  )
}
