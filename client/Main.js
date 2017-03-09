const React = require('react')
const h = require('react-hyperscript')
const createBrowserHistory = require('history').createBrowserHistory
const Router = require('react-router-dom').BrowserRouter
const Route = require('react-router-dom').Route
const Link = require('react-router-dom').Link

const CardsView = require('./CardsView')
const SiteDetail = require('./SiteDetail')
const UserAccount = require('./UserAccount')
const auth0 = require('./auth').auth0
const getLogoutURL = require('./auth').getLogoutURL
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

    auth0.parseHash(location.hash, (err, result) => {
      if (err) {
        console.log('error parsing hash:', err)
        setToken('')
        return
      }

      if (result) {
        setToken(result.idToken)
      }
    })
    location.hash = ''
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
              : h('a.nav-item', {onClick: this.login}, 'login'),
              this.state.isLogged
              ? h('a.nav-item', {onClick: this.logout}, 'logout')
              : h('a.nav-item', {onClick: this.login}, 'start tracking your sites!')
            ])
          ]),
          h(Route, {exact: true, path: '/sites', component: CardsView}),
          h(Route, {exact: true, path: '/sites/:code', component: SiteDetail}),
          h(Route, {exact: true, path: '/public/:code', component: SiteDetail}),
          h(Route, {exact: true, path: '/account', component: UserAccount})
        ])
      ])
    )
  },

  login (e) {
    e.preventDefault()
    auth0.authorize()
  },

  logout (e) {
    e.preventDefault()
    location.href = getLogoutURL()
  }
})
