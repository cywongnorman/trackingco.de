const React = require('react')
const h = require('react-hyperscript')
const findDOMNode = require('react-dom').findDOMNode
const TangleText = require('react-tangle')
const throttle = require('throttleit')

const snippet = require('./snippet')
const graphql = require('./graphql')
const formatdate = require('./helpers').formatdate
const onLoggedStateChange = require('./auth').onLoggedStateChange

const charts = {
  Main: require('./charts/Main'),
  SessionsByReferrer: require('./charts/SessionsByReferrer')
}

module.exports = React.createClass({
  getInitialState () {
    return {
      site: null,
      dataMax: 100,
      nlastdays: 45,
      sessionsLimit: 400,
      sessionsMinScore: 1,
      sessionsReferrerSelected: '',
      sessionsReferrerFilter: ''
    }
  },

  query (last) {
    last = last || this.state.nlastdays

    graphql.query(`
query d($code: String!, $last: Int, $l: Int, $s: Int, $r: String) {
  site(code: $code, last: $last) {
    name
    code
    created_at
    shareURL
    days {
      day
      s
      v
    }
    sessionsbyreferrer(limit: $l, minscore: $s, referrer: $r) {
      referrer
      scores
    }
    pages { a, c }
    referrers { a, c }
    today {
      s
      v
      b
    }
  }
}
    `, {
      code: this.props.match.params.code,
      last: last,
      l: this.state.sessionsLimit,
      s: this.state.sessionsMinScore,
      r: this.state.sessionsReferrerFilter
    })
    .then(r => {
      this.setState({
        site: r.site,
        nlastdays: last,
        dataMax: Math.max(...r.site.days.map(d => d.v))
      })
    })
    .catch(console.log.bind(console))
  },

  componentDidMount () {
    this.setSessionsLimit()
    window.addEventListener('resize', this.setSessionsLimit)

    onLoggedStateChange(isLogged => {
      if (isLogged) {
        this.query()
      }
    })
  },

  componentWillUnmount () {
    window.removeEventListener('resize', this.setSessionsLimit)
  },

  render () {
    let isOwner = this.props.location.pathname.split('/')[1] === 'sites'

    return this.state.site
    ? (
      h('.container', [
        h('.content', [
          h('h4.title.is-3', this.state.site.name),
          h('h6.subtitle.is-6', this.state.site.code)
        ]),
        this.state.dataMax === 0 && this.state.site.today.v === 0
        ? isOwner
          ? h(NoData, this.state)
          : ''
        : (
          h(Data, {
            ...this.state,
            isOwner: isOwner,
            updateNLastDays: this.query,
            updateMinScore: v => { this.setState({sessionsMinScore: v}, this.query) },
            updateSessionsReferrerSelected: data => {
              this.setState({sessionsReferrerSelected: data.payload.referrer})
            },
            filterByReferrer: this.filterByReferrer,
            dontFilterByReferrer: this.dontFilterByReferrer,
            toggleSharing: this.toggleSharing
          })
        )
      ])
    )
    : h('div')
  },

  toggleSharing (e) {
    e.preventDefault()
    graphql.mutate(`
      ($code: String!, $share: Boolean!) {
        shareSite(code: $code, share: $share) {
          ok
        }
      }
    `, {code: this.state.site.code, share: !this.state.site.shareURL})
    .then(r => {
      if (r.shareSite.ok) {
        this.query()
      } else {
        console.log('error setting site shared state.')
      }
    })
    .catch(e => {
      console.log(e.stack)
    })
  },

  setSessionsLimit: throttle(function (e) {
    let sessionsLimit = parseInt(findDOMNode(this).offsetWidth / 5)
    if (sessionsLimit === this.state.sessionsLimit) return

    this.setState({sessionsLimit: sessionsLimit}, this.query)
  }, 1000),

  filterByReferrer () {
    this.setState({
      sessionsReferrerSelected: '',
      sessionsReferrerFilter: this.state.sessionsReferrerSelected
    }, this.query)
  },

  dontFilterByReferrer () {
    this.setState({
      sessionsReferrerSelected: '',
      sessionsReferrerFilter: ''
    }, this.query)
  }
})

const Data = React.createClass({
  getInitialState () {
    return {
      pagesOpen: false,
      referrersOpen: false,
      showSnippet: false,
      showAbout: false
    }
  },

  render () {
    let pagesMore = this.props.site.pages.length > 12
    var pages = this.props.site.pages
    if (!this.state.pagesOpen) {
      pages = this.props.site.pages.slice(0, 12)
    }
    let referrersMore = this.props.site.referrers.length > 12
    var referrers = this.props.site.referrers
    if (!this.state.referrersOpen) {
      referrers = this.props.site.referrers.slice(0, 12)
    }

    let individualSessions = charts.SessionsByReferrer.sessionGroupsToIndividual(
        this.props.site.sessionsbyreferrer
    )

    let totalSessions = this.props.site.referrers.map(({c: count}) => count).reduce((a, b) => a + b, 0)

    return (
      h('.container', [
        h('.columns', [
          h('.column.is-third', [
            h('.card.detail-today.has-text-left', [
              h('.card-content', [
                h('h4.subtitle.is-4', 'Pageviews today:'),
                h('h1.title.is-1', this.props.site.today.v || 0)
              ])
            ])
          ]),
          h('.column.is-third', [
            h('.card.detail-today.has-text-centered', [
              h('.card-content', [
                h('h4.subtitle.is-4', 'Bounce rate today:'),
                h('h1.title.is-1',
                  typeof this.props.site.today.b === 'number'
                  ? (this.props.site.today.b * 100).toFixed(1) + '%'
                  : '-'
                )
              ])
            ])
          ]),
          h('.column.is-third', [
            h('.card.detail-today.has-text-right', [
              h('.card-content', [
                h('h4.subtitle.is-4', 'Sessions today:'),
                h('h1.title.is-1', this.props.site.today.s || 0)
              ])
            ])
          ])
        ]),
        h('.card.detail-chart-main', [
          h('.card-header', [
            h('.card-header-title', [
              'Number of sessions and pageviews',
              h(TangleChangeLastDays, this.props)
            ])
          ]),
          h('.card-image', [
            h('figure.image', [
              h(charts.Main, {
                site: this.props.site,
                dataMax: this.props.dataMax
              })
            ])
          ])
        ]),
        h('.card.detail-chart-individualsessions', [
          h('.card-header', [
            h('.card-header-title', [
              `showing ${individualSessions.length} sessions `,
              h(TangleChangeMinScore, this.props),
              ` from a total of ${totalSessions} `,
              h(TangleChangeLastDays, this.props)
            ])
          ]),
          h('.card-image', [
            h('figure.image', [
              h(charts.SessionsByReferrer, {
                site: this.props.site,
                individualSessions: individualSessions,
                handleClick: this.props.updateSessionsReferrerSelected
              })
            ])
          ]),
          h('.card-content', {style: {paddingTop: '3px', paddingBottom: '5px'}}, [
            h('.content', [
              h('p', this.props.sessionsReferrerFilter
                ? [
                  'seeing sessions from ',
                  h('b', this.props.sessionsReferrerFilter),
                  'only, ',
                  h('a', {onClick: this.props.dontFilterByReferrer}, 'view from all?')
                ]
                : this.props.sessionsReferrerSelected
                  ? [
                    'selected ',
                    h('b', this.props.sessionsReferrerSelected),
                    ', ',
                    h('a', {onClick: this.props.filterByReferrer}, 'see sessions from this referrer only?')
                  ]
                  : 'click at a session bar to selected its referrer.'
              )
            ])
          ])
        ]),
        h('.columns', [
          h('.column.is-half', [
            h('.card.detail-table', [
              h('.card-header', [
                h('.card-header-title', [
                  'Most viewed pages',
                  h(TangleChangeLastDays, this.props)
                ])
              ]),
              h('.card-content', [
                h('table.table', [
                  h('tbody', pages.map(({a: addr, c: count}) =>
                    h('tr', [
                      h('td', addr),
                      h('td', count)
                    ])
                  ))
                ]),
                pagesMore
                ? this.state.pagesOpen
                  ? h('a', {onClick: () => { this.setState({pagesOpen: false}) }}, 'see less')
                  : h('a', {onClick: () => { this.setState({pagesOpen: true}) }}, 'see more')
                : ''
              ])
            ])
          ]),
          h('.column.is-half', [
            h('.card.detail-table', [
              h('.card-header', [
                h('.card-header-title', [
                  'Top referring sites',
                  h(TangleChangeLastDays, this.props)
                ])
              ]),
              h('.card-content', [
                h('table.table', [
                  h('tbody', referrers.map(({a: addr, c: count}) =>
                    h('tr', [
                      h('td', [
                        addr + ' ',
                        addr === '<direct>'
                        ? ''
                        : (
                          h('a', {target: '_blank', href: addr}, [
                            h('img', {src: `data:image/svg+xml,<%3Fxml%20version%3D"1.0"%20encoding%3D"UTF-8"%20standalone%3D"no"%3F><svg%20xmlns%3D"http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg"%20width%3D"12"%20height%3D"12"><path%20fill%3D"%23fff"%20stroke%3D"%2306c"%20d%3D"M1.5%204.518h5.982V10.5H1.5z"%2F><path%20d%3D"M5.765%201H11v5.39L9.427%207.937l-1.31-1.31L5.393%209.35l-2.69-2.688%202.81-2.808L4.2%202.544z"%20fill%3D"%2306f"%2F><path%20d%3D"M9.995%202.004l.022%204.885L8.2%205.07%205.32%207.95%204.09%206.723l2.882-2.88-1.85-1.852z"%20fill%3D"%23fff"%2F><%2Fsvg>`})
                          ])
                        )
                      ]),
                      h('td', count)
                    ])
                  ))
                ]),
                referrersMore
                ? this.state.referrersOpen
                  ? h('a', {onClick: () => { this.setState({referrersOpen: false}) }}, 'see less')
                  : h('a', {onClick: () => { this.setState({referrersOpen: true}) }}, 'see more')
                : ''
              ])
            ])
          ])
        ]),
        h('.columns', [
          h('.column.is-half', [
            h('.card.detail-about', [
              h('.card-header', [
                h('p.card-header-title', 'Site information'),
                h('a.card-header-icon', {
                  onClick: () => { this.setState({showAbout: !this.state.showAbout}) }
                }, [
                  h('span.icon', [
                    h(`i.fa.fa-angle-${this.state.showAbout ? 'down' : 'up'}`)
                  ])
                ])
              ]),
              this.state.showAbout
              ? (
                h('.card-content', [
                  h('aside.menu', [
                    h('p.menu-label', 'Name'),
                    h('ul.menu-list', [
                      h('li', this.props.site.name)
                    ]),
                    h('p.menu-label', 'Code'),
                    h('ul.menu-list', [
                      h('li', this.props.site.code)
                    ]),
                    h('p.menu-label', 'Sharing'),
                    h('ul.menu-list', [
                      h('li', this.props.site.shareURL
                        ? 'This site is public, you can share it the following URL:'
                        : 'This site is private'
                      ),
                      this.props.site.shareURL && h('li', [
                        h('input.input', {
                          disabled: true,
                          value: this.props.site.shareURL,
                          style: {marginTop: '4px'}
                        })
                      ]),
                      this.props.isOwner &&
                      h('li', [
                        h('a.button.is-warning.is-small.is-inverted.is-outlined', {
                          style: {display: 'inline-block'},
                          onClick: this.props.toggleSharing
                        }, this.props.site.shareURL
                          ? 'Make it private'
                          : 'Make it public and share'
                        )
                      ])
                    ]),
                    h('p.menu-label', 'Creation date'),
                    h('ul.menu-list', [
                      h('li', formatdate(this.props.site.created_at))
                    ])
                  ])
                ])
              )
              : ''
            ])
          ]),
          this.props.isOwner && h('.column.is-half', [
            h('.card.detail-trackingcode', [
              h('.card-header', [
                h('p.card-header-title', 'Tracking code'),
                h('a.card-header-icon', {
                  onClick: () => { this.setState({showSnippet: !this.state.showSnippet}) }
                }, [
                  h('span.icon', [
                    h(`i.fa.fa-angle-${this.state.showSnippet ? 'down' : 'up'}`)
                  ])
                ])
              ]),
              this.state.showSnippet
              ? (
                h('.card-content', [
                  h('.content', [
                    h('p', 'Paste the following in any part of your site:'),
                    h('pre', [
                      h('code', `<script>;${snippet(this.props.site.code)};</script>`)
                    ])
                  ])
                ])
              )
              : ''
            ])
          ])
        ])
      ])
    )
  }
})

const NoData = function (props) {
  return (
    h('.card.detail-trackingcode', [
      h('.card-content', [
        h('.content', [
          h('p', 'This site has no data yet. Have you installed the tracking code?'),
          h('p', 'Just paste the following in any part of your site:'),
          h('pre', [
            h('code', `<script>;${snippet(props.site.code)};</script>`)
          ])
        ])
      ])
    ])
  )
}

const TangleChangeLastDays = function (props) {
  return (
    h('small', [
      'in the last ',
      h(TangleText, {
        value: props.nlastdays,
        onChange: props.updateNLastDays,
        pixelDistance: 15,
        min: 2,
        max: 90
      }),
      ' days'
    ])
  )
}

const TangleChangeMinScore = function (props) {
  return (
    h('small', [
      'with minimum ',
      h(TangleText, {
        value: props.sessionsMinScore,
        onChange: props.updateMinScore,
        pixelDistance: 40,
        min: 1,
        max: 99
      }),
      ' score'
    ])
  )
}
