// We'll use React without JSX to avoid setting up Webpack and Babel.
 
const e = (el, props, children) => {
  if (props) {
    const { cls, ...rest } = props;
    return React.createElement(el, { ...rest, className: cls }, children);
  } else {
    return React.createElement(el, null, children);
  }
}

function ActionButton({ disabled, action, active }) {
   return e('div', { cls: 'content' }, [
     e('div', { 
      cls: `button ${active ? 'is-danger' : 'is-success' }`, 
      onClick: action, 
      disabled
    }, active ? 'Stop' : 'Start')
   ]);
}

function startSession(offer) {
  return fetch('/session', {
    method: 'POST',
    body: JSON.stringify({
      offer
    }),
    headers: {
      'Content-Type': 'application/json'
    }
  }).then(res => {
    return res.json();
  }).then(msg => {
    return msg.answer;
  });
}

function setupPeerConnection({ stream, onResult, onSignaling, onStop }) {
  const pc = new RTCPeerConnection({
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
  });
  const resChan = pc.createDataChannel('results', {
    ordered: true,
    protocol: 'tcp'
  });
  resChan.onmessage = evt => {
    // evt.data will be an instance of ArrayBuffer
    const dec = new TextDecoder('utf-8');
    const strData = dec.decode(evt.data)
    const result = JSON.parse(strData);
    onResult(result);
  };

  // We close everything when the data channel closes
  resChan.onclose = () => {
    pc.close();
    onStop()
  };

  pc.onicecandidate = evt => {
    if (!evt.candidate) {
      // ICE Gathering finished 
      const { sdp: offer } = pc.localDescription;
      startSession(offer).then(answer => {
        onSignaling(offer, answer);
        const rd = new RTCSessionDescription({
          sdp: answer,
          type: 'answer'
        });
        pc.setRemoteDescription(rd);
      });
    }
  };

  const audioTracks = stream.getAudioTracks();
  if (audioTracks.length > 0) {
    pc.addTrack(audioTracks[0], stream);
  }
  // Let's trigger ICE gathering
  pc.createOffer({
    offerToReceiveAudio: false,
    offerToReceiveVideo: false
  }).then(ld => {
    pc.setLocalDescription(ld)
  });
  return pc;
}

function Results({ results }) {
  if (!results.length) {
    return e('p', { cls: 'has-text-centered' }, 'No results yet');
  }
  return e('div', { cls: 'content' },
    results.map(r =>
      e('p', null, [
        'Confidence: ', e('strong', null, (r.confidence * 100).toFixed(1) + '%'),
        '. Result: ', e('strong', null, r.text), '.',
      ])
    )
  );
}

const initialState = {
  pc: null,
  stream: null,
  offer: null,
  answer: null,
  error: null,
  results: [],
  active: false
};

function AppContent() {
  const [state, setState] = React.useState(initialState);

  function start() {
    setState(st => ({ ...st, offer: null, answer: null, error: null }));

    navigator.mediaDevices.getUserMedia({
      audio: true,
      video: false
    }).then(stream => {
      const pc = setupPeerConnection({
        stream, 
        onSignaling: (offer, answer) => setState(st => ({ ...st, offer, answer })),
        onResult: (r) => setState(st => ({ ...st, results: [...st.results, r ] })),
        onStop: () => setState(st => ({ ...st, pc: null })),
      });

      setState(st => ({ ...st, stream, pc, active: true }));
    }).catch(error => {
      setState(st => ({ ...st, error }));
    });
  }

  function stop() {
    state.stream && state.stream.getAudioTracks().forEach(tr => tr.stop());
    setState(st => ({ ...state, stream: null, active: false }));
  }

  const action = state.active ? stop: start;
  return e('div', { cls: 'box is-radiusless' }, [
    e(ActionButton, { active: state.active, action, disabled: (!!state.pc) && !state.active }),
    e('h3', { cls: 'subtitle'}, 'Results'),
    e(Results, { results: state.results }),
    e('h3', { cls: 'subtitle'}, 'Offer'),
    e('pre', { cls: 'is-family-code' }, state.offer || '-'),
    e('h3', { cls: 'subtitle'}, 'Answer'),
    e('pre', { cls: 'is-family-code' }, state.answer || '-'),
  ])
}

function App() {
  return e('section', { cls: 'section'}, [
    e('div', { cls: 'container' }, [
      e('h1', { cls: 'title'}, 'WebRTC speech to text'),
      e('p', { cls: 'subtitle'}, 'Powered by Go and Pion WebRTC'),
      e(AppContent)
    ])
  ]);
}

document.addEventListener('DOMContentLoaded', () => {
  ReactDOM.render(e(App), document.getElementById('app'));
});