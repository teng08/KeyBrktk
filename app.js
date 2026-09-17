const sounds = {
  tactile: { name: 'Tactile Brown', click: 1250, body: 155, clickGain: 0.12, bodyGain: 0.24, decay: 0.065 },
  clicky: { name: 'Clicky Blue', click: 2450, body: 210, clickGain: 0.2, bodyGain: 0.16, decay: 0.045 },
  thocky: { name: 'Deep Thock', click: 540, body: 92, clickGain: 0.045, bodyGain: 0.34, decay: 0.11 },
  typewriter: { name: 'Typewriter', click: 1750, body: 280, clickGain: 0.17, bodyGain: 0.2, decay: 0.05 }
};
let selectedSound = 'tactile';
let room = 'small';
let volume = 0.72;
let muted = false;
let keystrokes = 0;
let audioContext;
let masterGain;

const getAudio = () => {
  if (!audioContext) {
    audioContext = new (window.AudioContext || window.webkitAudioContext)();
    masterGain = audioContext.createGain();
    masterGain.gain.value = volume;
    masterGain.connect(audioContext.destination);
  }
  if (audioContext.state === 'suspended') audioContext.resume();
};

function noiseBurst(time, duration, gainAmount, filterFrequency) {
  const buffer = audioContext.createBuffer(1, audioContext.sampleRate * duration, audioContext.sampleRate);
  const data = buffer.getChannelData(0);
  for (let index = 0; index < data.length; index += 1) data[index] = Math.random() * 2 - 1;
  const source = audioContext.createBufferSource();
  const filter = audioContext.createBiquadFilter();
  const gain = audioContext.createGain();
  source.buffer = buffer;
  filter.type = 'bandpass'; filter.frequency.value = filterFrequency; filter.Q.value = 1.2;
  gain.gain.setValueAtTime(gainAmount, time); gain.gain.exponentialRampToValueAtTime(0.001, time + duration);
  source.connect(filter).connect(gain).connect(masterGain);
  source.start(time); source.stop(time + duration);
}

function playKey(keyType = 'normal') {
  getAudio();
  const profile = sounds[selectedSound];
  if (muted) return;
  const now = audioContext.currentTime;
  const multiplier = keyType === 'space' ? 1.45 : keyType === 'shift' ? 1.18 : 1;
  noiseBurst(now, profile.decay * multiplier, profile.clickGain * multiplier, profile.click);
  const oscillator = audioContext.createOscillator();
  const gain = audioContext.createGain();
  oscillator.type = selectedSound === 'thocky' ? 'sine' : 'triangle';
  oscillator.frequency.setValueAtTime(profile.body, now);
  oscillator.frequency.exponentialRampToValueAtTime(Math.max(45, profile.body * .65), now + .13);
  gain.gain.setValueAtTime(profile.bodyGain * multiplier, now);
  gain.gain.exponentialRampToValueAtTime(0.001, now + .13 * multiplier);
  oscillator.connect(gain).connect(masterGain); oscillator.start(now); oscillator.stop(now + .14);
  if (room !== 'small') noiseBurst(now + .01, room === 'hall' ? .28 : .16, room === 'hall' ? .035 : .02, room === 'hall' ? 700 : 1100);
  keystrokes += 1;
  document.getElementById('keystrokeCount').textContent = `${keystrokes} keystroke${keystrokes === 1 ? '' : 's'} made`;
}

function pressVisual(keyName) {
  const key = document.querySelector(`[data-key="${keyName}"]`);
  if (!key) return;
  key.classList.add('pressed');
  window.setTimeout(() => key.classList.remove('pressed'), 110);
}

document.querySelectorAll('.key').forEach((key) => key.addEventListener('click', () => {
  const type = key.dataset.key === 'SPACE' ? 'space' : key.dataset.key === 'SHIFT' ? 'shift' : 'normal';
  playKey(type); pressVisual(key.dataset.key);
}));

document.addEventListener('keydown', (event) => {
  if (event.repeat) return;
  const keyName = event.key === ' ' ? 'SPACE' : event.key.toUpperCase();
  const allowed = ['Q','W','E','R','T','Y','U','I','O','P','A','S','D','F','G','H','J','K','L','Z','X','C','V','B','N','M','BACKSPACE','ENTER','SHIFT','SPACE'];
  if (!allowed.includes(keyName)) return;
  event.preventDefault();
  const type = keyName === 'SPACE' ? 'space' : keyName === 'SHIFT' ? 'shift' : 'normal';
  playKey(type); pressVisual(keyName);
});

document.querySelectorAll('.sound-card').forEach((card) => card.addEventListener('click', () => {
  selectedSound = card.dataset.sound;
  document.querySelectorAll('.sound-card').forEach((item) => item.classList.toggle('active', item === card));
  document.getElementById('currentSound').textContent = sounds[selectedSound].name;
  document.getElementById('statusText').textContent = `${sounds[selectedSound].name.toUpperCase()} LOADED`;
  playKey();
}));

document.getElementById('volumeSlider').addEventListener('input', (event) => {
  volume = event.target.value / 100;
  document.getElementById('volumeValue').textContent = `${event.target.value}%`;
  if (masterGain) masterGain.gain.value = muted ? 0 : volume;
});

document.querySelectorAll('.room-button').forEach((button) => button.addEventListener('click', () => {
  room = button.dataset.room;
  document.querySelectorAll('.room-button').forEach((item) => item.classList.toggle('active', item === button));
  document.getElementById('roomValue').textContent = room === 'small' ? 'Small studio' : room === 'desk' ? 'Wood desk' : 'Open hall';
}));

document.getElementById('muteButton').addEventListener('click', () => {
  muted = !muted;
  if (masterGain) masterGain.gain.value = muted ? 0 : volume;
  document.getElementById('muteButton').textContent = muted ? '◗' : '◖';
  document.getElementById('statusText').textContent = muted ? 'MUTED' : 'READY TO TYPE';
});
