module.exports = {
  info () {
    window.notie.alert({
      text: Array.prototype.join.call(arguments, ' '),
      type: 'info',
      time: 3
    })
    console.log.apply(console, arguments)
  },

  debug () {
    console.debug.apply(console, arguments)
  },

  error (e) {
    if (e.stack) {
      console.error(e.stack)
      window.notie.alert({
        text: 'Something wrong has occurred, see the console for the complete error.',
        type: 'error',
        time: 3
      })
      return
    }

    window.notie.alert({
      text: Array.prototype.join.call(arguments, ' '),
      type: 'error',
      time: 5
    })
    console.error.apply(console, arguments)
  },

  success () {
    window.notie.alert({
      text: Array.prototype.join.call(arguments, ' '),
      type: 'success',
      time: 4
    })
  },

  confirm (text, confirmed) {
    window.notie.confirm({text}, confirmed)
  }
}
