const React = require('react')
const h = require('react-hyperscript')
const page = require('page')
const findDOMNode = require('react-dom').findDOMNode
const TangleText = require('react-tangle-text')
const RIEInput = require('riek').RIEInput
const throttle = require('throttleit')
const DocumentTitle = require('react-document-title')
const BodyStyle = require('body-style')
const urlTrie = require('url-trie')
const reduceObject = require('just-reduce-object')

const log = require('./log')
const snippet = require('./snippet')
const graphql = require('./graphql')
const formatdate = require('./helpers').formatdate
const coloursfragment = require('./helpers').coloursfragment
const mergeColours = require('./helpers').mergeColours
const title = require('./helpers').title
const onLoggedStateChange = require('./auth').onLoggedStateChange

const charts = {
  MainDays: require('./charts/MainDays'),
  MainMonths: require('./charts/MainMonths'),
  SessionsByReferrer: require('./charts/SessionsByReferrer')
}

const SiteDetail = React.createClass({
  getInitialState () {
    return {
      site: null,
      dataMax: 100,
      nlastdays: 45,
      usingMonths: false,
      sessionsLimit: 400,
      sessionsMinScore: 1,
      sessionsReferrerSelected: undefined,
      sessionsReferrerFilter: undefined
    }
  },

  query (last) {
    last = last || this.state.nlastdays
    let usingMonths = last > 90

    let daysfields = `
    days { day, s, v }
    sessionsbyreferrer(limit: $l, minscore: $s, referrer: $r) {
      referrer, scores
    }
    `

    let monthsfields = 'months { month, v, s, b, c }'

    graphql.query(`
query d($code: String!, $last: Int${usingMonths ? '' : ', $l: Int, $s: Int, $r: String'}) {
  site(code: $code, last: $last) {
    name
    code
    created_at
    shareURL
    ${usingMonths ? monthsfields : daysfields}
    pages { a, c }
    referrers { a, c }
    today { s, v, b }
  }
  me {
    colours { ...${coloursfragment} }
    domains
  }
}
    `, {
      code: this.props.code,
      last: last,
      l: this.state.sessionsLimit,
      s: this.state.sessionsMinScore,
      r: this.state.sessionsReferrerFilter
    })
    .then(r => {
      this.setState({
        site: r.site,
        me: r.me,
        nlastdays: last,
        usingMonths,
        dataMax: Math.max(
          ...(r.site.days || []).map(d => d.v),
          ...(r.site.months || []).map(d => d.v)
        )
      })
    })
    .catch(log.error)
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
    if (!this.state.site) {
      return h('div')
    }

    let colours = mergeColours(this.state.me.colours)

    return (
      h('.container', [
        h('.content', [
          h('h4.title.is-3', this.state.site.name),
          h('h6.subtitle.is-6', this.state.site.code)
        ]),
        this.state.dataMax === 0 && this.state.site.today.v === 0
          ? (
            h(NoData, {
              ...this.state,
              colours,
              isOwner: true,
              toggleSharing: this.toggleSharing,
              confirmDelete: this.confirmDelete,
              confirmRename: this.confirmRename
            })
          )
          : (
            h(Data, {
              ...this.state,
              colours,
              isOwner: true,
              updateNLastDays: this.query,
              updateMinScore: v => { this.setState({sessionsMinScore: v}, this.query) },
              updateSessionsReferrerSelected: data => {
                this.setState({sessionsReferrerSelected: data.payload.referrer})
              },
              filterByReferrer: this.filterByReferrer,
              dontFilterByReferrer: this.dontFilterByReferrer,
              toggleSharing: this.toggleSharing,
              confirmDelete: this.confirmDelete,
              confirmRename: this.confirmRename
            })
          )
      ])
    )
  },

  toggleSharing (e) {
    e.preventDefault()

    graphql.mutate(`
($code: String!, $share: Boolean!) {
  shareSite(code: $code, share: $share) {
    ok, error
  }
}
    `, {code: this.state.site.code, share: !this.state.site.shareURL})
    .then(r => {
      if (!r.shareSite.ok) {
        log.error('error setting shared state:', r.shareSite.error)
        return
      }
      log.info(`This site is now ${!this.state.site.shareURL ? '' : 'un'}shared.`)
      this.query()
    })
    .catch(log.error)
  },

  confirmRename ({name}) {
    graphql.mutate(`
      ($name: String!, $code: String!) {
        renameSite(name: $name, code: $code) {
          name
        }
      }
    `, {name, code: this.state.site.code})
    .then(r => {
      log.info(`${this.state.site.name} renamed to ${r.renameSite.name}.`)
      this.setState(st => {
        st.site.name = r.renameSite.name
        return st
      })
    })
    .catch(log.error)
  },

  confirmDelete () {
    graphql.mutate(`
($code: String!) {
  deleteSite(code: $code) {
    ok, error
  }
}
    `, {code: this.state.site.code})
    .then(r => {
      if (!r.deleteSite.ok) {
        log.error('failed to delete site:', r.deleteSite.error)
        this.setState({deleting: false})
        return
      }
      log.info(`${this.state.site.name} was deleted.`)
      page('/sites')
    })
    .catch(log.error)
  },

  setSessionsLimit: throttle(function (e) {
    let sessionsLimit = parseInt(findDOMNode(this).offsetWidth / 5)
    if (sessionsLimit === this.state.sessionsLimit) return

    this.setState({sessionsLimit: sessionsLimit}, this.query)
  }, 1000),

  filterByReferrer () {
    this.setState({
      sessionsReferrerSelected: undefined,
      sessionsReferrerFilter: this.state.sessionsReferrerSelected
    }, this.query)
  },

  dontFilterByReferrer () {
    this.setState({
      sessionsReferrerSelected: undefined,
      sessionsReferrerFilter: undefined
    }, this.query)
  }
})

const Data = React.createClass({
  getInitialState () {
    return {
      pagesOpen: false,
      referrersOpen: false,
      referrersTrie: null,
      showSnippet: false
    }
  },

  render () {
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

    referrers.sort((a, b) => (b.countdeep + b.counthere) - (a.countdeep + a.counthere))

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
      h(DocumentTitle, {title: title(this.props.site.name)}, [
        h(BodyStyle, {style: {backgroundColor: this.props.colours.background}}, [
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
                  this.props.usingMonths
                    ? h(charts.MainMonths, {
                      site: this.props.site,
                      dataMax: this.props.dataMax,
                      colours: this.props.me.colours
                    })
                    : h(charts.MainDays, {
                      site: this.props.site,
                      dataMax: this.props.dataMax,
                      colours: this.props.me.colours
                    })
                ])
              ])
            ]),
            !this.props.usingMonths && (
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
                    h('p', this.props.sessionsReferrerFilter !== undefined
                      ? [
                        'seeing sessions from ',
                        h('b', this.props.sessionsReferrerFilter),
                        'only, ',
                        h('a', {onClick: this.props.dontFilterByReferrer}, 'view from all?')
                      ]
                      : this.props.sessionsReferrerSelected !== undefined
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
              ])
            ) || null,
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
                      this.state.referrersTrie && h('thead', [
                        h('tr', [
                          h('td', {colSpan: 3, style: {textAlign: 'right'}}, [
                            h('a', {
                              onClick: e => {
                                e.preventDefault()
                                let prev = referrersTrie.prev
                                delete referrersTrie.prev
                                this.setState({referrersTrie: prev})
                              }
                            }, '↥')
                          ])
                        ])
                      ]),
                      h('tbody', referrers.map(({addr, counthere, countdeep, href, more}) =>
                        h('tr', [
                          h('td', [
                            addr + ' ',
                            href && addr !== '<direct>' && (
                              h('a', {target: '_blank', href}, [
                                h('img', {src: `data:image/svg+xml,<%3Fxml%20version%3D"1.0"%20encoding%3D"UTF-8"%20standalone%3D"no"%3F><svg%20xmlns%3D"http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg"%20width%3D"12"%20height%3D"12"><path%20fill%3D"%23fff"%20stroke%3D"%2306c"%20d%3D"M1.5%204.518h5.982V10.5H1.5z"%2F><path%20d%3D"M5.765%201H11v5.39L9.427%207.937l-1.31-1.31L5.393%209.35l-2.69-2.688%202.81-2.808L4.2%202.544z"%20fill%3D"%2306f"%2F><path%20d%3D"M9.995%202.004l.022%204.885L8.2%205.07%205.32%207.95%204.09%206.723l2.882-2.88-1.85-1.852z"%20fill%3D"%23fff"%2F><%2Fsvg>`})
                              ])
                            )
                          ]),
                          h('td', counthere),
                          h('td', more && [
                            h('a', {
                              onClick: e => {
                                e.preventDefault()
                                more.prev = this.state.referrersTrie
                                this.setState({referrersTrie: more})
                              },
                              title: `other ${countdeep} URL${countdeep !== 1 ? 's in paths' : ' in a path'} after this`
                            }, `↦ ${countdeep}`)
                          ])
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
                h(About, this.props)
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
                  this.state.showSnippet && h('.card-content', [
                    h('.content', [
                      h('p', 'Paste the following in any part of your site:'),
                      h('pre', [
                        h('code',
                          `<script>;${snippet(this.props.site.code, this.props.me.domains[0])};</script>`)
                      ])
                    ])
                  ])
                ])
              ])
            ])
          ])
        ])
      ])
    )
  }
})

const About = React.createClass({
  getInitialState () {
    return {
      showAbout: false,
      isRenaming: false,
      isDeleting: false
    }
  },

  render () {
    return (
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
        this.state.showAbout &&
          h('.card-content', [
            h('aside.menu', [
              h('p.menu-label', 'Name'),
              h('ul.menu-list', [
                h('li', [
                  h(RIEInput, {
                    value: this.props.site.name,
                    propName: 'name',
                    change: this.props.confirmRename,
                    shouldBlockWhileLoading: true,
                    classEditing: 'input',
                    classLoading: 'is-warning'
                  })
                ])
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
              ].concat(
                this.state.isDeleting
                  ? [
                    h('li.delete-site.cancel', [
                      h('a.button.is-large', {
                        onClick: e => {
                          e.preventDefault()
                          this.setState({isDeleting: false})
                        }
                      }, 'do not delete')
                    ]),
                    h('li.delete-site.confirm', [
                      h('a.button.is-danger.is-small', {
                        onClick: e => {
                          e.preventDefault()
                          this.props.confirmDelete()
                          this.setState({isDeleting: false})
                        }
                      }, 'delete irrecoverably')
                    ])
                  ]
                  : (
                    h('li.delete-site.start', [
                      h('a.button.is-danger.is-small', {
                        style: {display: 'inline-block'},
                        onClick: e => {
                          e.preventDefault()
                          this.setState({isDeleting: true})
                        }
                      }, 'delete site')
                    ])
                  )
              ))
            ])
          ])
      ])
    )
  }
})

const NoData = React.createClass({
  defaultDomain: 't.trackingco.de',

  getInitialState () {
    return {
      domain: this.defaultDomain
    }
  },

  componentWillMount () {
    if (this.props.me.domains.length) {
      this.setState({domain: this.props.me.domains[0]})
    }
  },

  render () {
    return (
      h(DocumentTitle, {title: title(this.props.site.name)}, [
        h(BodyStyle, {style: {backgroundColor: this.props.colours.background}}, [
          h('.container', [
            h('.columns', [
              h('.column.is-full', [
                h('.card.detail-trackingcode', [
                  h('.card-content', [
                    h('.content', [
                      h('p', 'This site has no data yet. Have you installed the tracking code?'),
                      h('p', 'Just paste the following in any part of your site:'),
                      h('pre', [
                        h('code', `<script>;${snippet(this.props.site.code, this.state.domain)};</script>`)
                      ]),
                      h('.level', {style: {marginTop: '14px'}}, [
                        h('.level-left', [
                          'Use a different domain: ',
                          h('span.select', {style: {marginLeft: '11px'}}, [
                            h('select', {
                              onChange: e => this.setState({domain: e.target.value}),
                              value: this.state.domain
                            }, this.props.me.domains.concat(this.defaultDomain).map(hostname =>
                              h('option', hostname)
                            ))
                          ])
                        ])
                      ])
                    ])
                  ])
                ]),
                h(About, this.props)
              ])
            ])
          ])
        ])
      ])
    )
  }
})

const TangleChangeLastDays = function (props) {
  return (
    h('small', [
      'in the last ',
      h(TangleText, {
        value: props.nlastdays,
        onChange: props.updateNLastDays,
        pixelDistance: 15,
        min: 1
      }),
      ' day' + (props.nlastdays === 1 ? '' : 's')
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

module.exports = SiteDetail
