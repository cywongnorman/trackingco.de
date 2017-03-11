const React = require('react')
const h = require('react-hyperscript')
const createBrowserHistory = require('history').createBrowserHistory
const Router = require('react-router-dom').BrowserRouter
const Route = require('react-router-dom').Route
const Link = require('react-router-dom').Link

const log = require('./log')
const CardsView = require('./CardsView')
const SiteDetail = require('./SiteDetail')
const UserAccount = require('./UserAccount')
const auth0 = require('./auth').auth0
const setToken = require('./auth').setToken
const onLoggedStateChange = require('./auth').onLoggedStateChange

module.exports = React.createClass({
  getInitialState () {
    return {
      isLogged: false
    }
  },

  componentDidMount () {
    onLoggedStateChange(isLogged => {
      this.setState({isLogged})
    })

    if (location.hash && location.hash.indexOf('token') !== -1) {
      auth0.parseHash(location.hash, (err, result) => {
        if (err) {
          log.error("error parsing account credentials, you'll be logged out.")
          log.debug(err)
          setToken('')
          return
        }

        log.success("You're now logged in!")
        setToken(result.idToken || result.id_token)
        location.hash = ''
      })
    }
  },

  render () {
    return (
      h(Router, {history: createBrowserHistory()}, [
        h('div', [
          h('nav.nav', [
            h('.nav-left', [
              h('a.nav-item', [
                h('img', {src: '/favicon.ico', alt: 'trackingcode logo'})
              ]),
              h('a.nav-item', 'trackingco.de')
            ]),
            h('.nav-center', [
              this.state.isLogged
              ? h(Link, {className: 'nav-item', to: '/account'}, 'account')
              : '',
              this.state.isLogged
              ? h(Link, {className: 'nav-item', to: '/sites'}, 'sites')
              : h('a.nav-item', {href: auth0.getLoginURL()}, 'login'),
              this.state.isLogged
              ? h('a.nav-item', {key: 'logout', onClick: auth0.logout}, 'logout')
              : h('a.nav-item', {key: 'login', href: auth0.getLoginURL()}, 'start tracking your sites!')
            ])
          ]),
          h(Route, {exact: true, path: '/sites', component: CardsView}),
          h(Route, {exact: true, path: '/sites/:code', component: SiteDetail}),
          h(Route, {exact: true, path: '/public/:code', component: SiteDetail}),
          h(Route, {exact: true, path: '/account', component: UserAccount})
        ])
      ])
    )
  }
})
