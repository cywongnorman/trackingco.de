const React = require('react')
const h = require('react-hyperscript')
const createBrowserHistory = require('history').createBrowserHistory
const Router = require('react-router-dom').BrowserRouter
const Route = require('react-router-dom').Route
const Link = require('react-router-dom').Link

const CardsView = require('./CardsView')
const SiteDetail = require('./SiteDetail')

module.exports = React.createClass({
  getInitialState () {
    return {
      menuActive: false
    }
  },

  render () {
    return (
      h(Router, {history: createBrowserHistory()}, [
        h('div', [
          h('nav.nav', [
            h('.nav-left', [
              h('a.nav-item', 'trackingco.de')
            ]),
            h('.nav-center', [
              h(Link, {className: 'nav-item', to: '/sites'}, 'your sites')
            ])
            // h('.nav-right.nav-menu', {className: menuActive ? 'is-active' : ''}, [
            //
            // ])
          ]),
          h(Route, {exact: true, path: '/sites', component: CardsView}),
          h(Route, {exact: true, path: '/sites/:code', component: SiteDetail}),
          h(Route, {exact: true, path: '/public/:code', component: SiteDetail})
        ])
      ])
    )
  }
})
