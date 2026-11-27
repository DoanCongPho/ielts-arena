// Traditional Christmas carols, all long in the public domain, written out
// as melody + one chord per bar for the in-browser music box
// (lib/carolPlayer.js). Nothing is downloaded: the player synthesises them.
//
// melody: space-separated "<note>:<beats>" tokens, a quarter note = 1 beat,
//         "R" for a rest. The first `pickup` beats come before bar 1.
// chords: one chord per bar (C, Dm, D7, …), played as an oom-pah
//         accompaniment underneath.

export const CAROLS = [
  {
    id: 'jingle-bells',
    title: 'Jingle Bells',
    bpm: 132,
    beatsPerBar: 4,
    pickup: 0,
    transpose: 0,
    sleighBells: true,
    melody: [
      'E5:1 E5:1 E5:2', 'E5:1 E5:1 E5:2', 'E5:1 G5:1 C5:1.5 D5:0.5', 'E5:4',
      'F5:1 F5:1 F5:1.5 F5:0.5', 'F5:1 E5:1 E5:1 E5:0.5 E5:0.5', 'E5:1 D5:1 D5:1 E5:1', 'D5:2 G5:2',
      'E5:1 E5:1 E5:2', 'E5:1 E5:1 E5:2', 'E5:1 G5:1 C5:1.5 D5:0.5', 'E5:4',
      'F5:1 F5:1 F5:1.5 F5:0.5', 'F5:1 E5:1 E5:1 E5:0.5 E5:0.5', 'G5:1 G5:1 F5:1 D5:1', 'C5:4',
    ].join(' '),
    chords: 'C C C C F C D7 G C C C C F C G C',
  },
  {
    id: 'silent-night',
    title: 'Silent Night',
    bpm: 84,
    beatsPerBar: 3,
    pickup: 0,
    transpose: 12,
    melody: [
      'G4:1.5 A4:0.5 G4:1', 'E4:3', 'G4:1.5 A4:0.5 G4:1', 'E4:3',
      'D5:2 D5:1', 'B4:3', 'C5:2 C5:1', 'G4:3',
      'A4:2 A4:1', 'C5:1.5 B4:0.5 A4:1', 'G4:1.5 A4:0.5 G4:1', 'E4:3',
      'A4:2 A4:1', 'C5:1.5 B4:0.5 A4:1', 'G4:1.5 A4:0.5 G4:1', 'E4:3',
      'D5:2 D5:1', 'F5:1.5 D5:0.5 B4:1', 'C5:3', 'E5:3',
      'C5:1.5 G4:0.5 E4:1', 'G4:1.5 F4:0.5 D4:1', 'C4:6',
    ].join(' '),
    chords: 'C C C C G G C C F F C C F F C C G G C C C G C C',
  },
  {
    id: 'we-wish-you',
    title: 'We Wish You a Merry Christmas',
    bpm: 150,
    beatsPerBar: 3,
    pickup: 1,
    transpose: 0,
    melody: [
      'G4:1',
      'C5:1 C5:0.5 D5:0.5 C5:0.5 B4:0.5', 'A4:1 A4:1 A4:1', 'D5:1 D5:0.5 E5:0.5 D5:0.5 C5:0.5', 'B4:1 G4:1 G4:1',
      'E5:1 E5:0.5 F5:0.5 E5:0.5 D5:0.5', 'C5:1 A4:1 G4:0.5 G4:0.5', 'A4:1 D5:1 B4:1', 'C5:2 G4:1',
      'C5:1 C5:1 C5:1', 'B4:2 B4:1', 'C5:1 B4:1 A4:1', 'G4:2 D5:1',
      'E5:1 D5:1 C5:1', 'G5:1 G4:1 G4:0.5 G4:0.5', 'A4:1 D5:1 B4:1', 'C5:3',
    ].join(' '),
    chords: 'C F D7 G Am F G C C G F G C C G C',
  },
  {
    id: 'deck-the-halls',
    title: 'Deck the Halls',
    bpm: 132,
    beatsPerBar: 4,
    pickup: 0,
    transpose: 0,
    melody: [
      'G5:1.5 F5:0.5 E5:1 D5:1', 'C5:1 D5:1 E5:1 C5:1', 'D5:0.5 E5:0.5 F5:0.5 D5:0.5 E5:1.5 D5:0.5', 'C5:1 B4:1 C5:2',
      'G5:1.5 F5:0.5 E5:1 D5:1', 'C5:1 D5:1 E5:1 C5:1', 'D5:0.5 E5:0.5 F5:0.5 D5:0.5 E5:1.5 D5:0.5', 'C5:1 B4:1 C5:2',
      'D5:1.5 E5:0.5 F5:1 D5:1', 'E5:1.5 F5:0.5 G5:1 D5:1', 'E5:0.5 F#5:0.5 G5:1 A5:0.5 B5:0.5 C6:1', 'B5:1 A5:1 G5:2',
      'G5:1.5 F5:0.5 E5:1 D5:1', 'C5:1 D5:1 E5:1 C5:1', 'A5:0.5 A5:0.5 A5:0.5 A5:0.5 G5:1.5 F5:0.5', 'E5:1 D5:1 C5:2',
    ].join(' '),
    chords: 'C C G C C C G C G C D G C C F C',
  },
  {
    id: 'joy-to-the-world',
    title: 'Joy to the World',
    bpm: 100,
    beatsPerBar: 2,
    pickup: 0,
    transpose: 12,
    melody: [
      'C5:1 B4:0.75 A4:0.25', 'G4:1.5 F4:0.5', 'E4:1 D4:1', 'C4:1.5 G4:0.5',
      'A4:1.5 A4:0.5', 'B4:1.5 B4:0.5', 'C5:1.5 C5:0.5', 'C5:0.5 B4:0.5 A4:0.5 G4:0.5',
      'G4:0.75 F4:0.25 E4:0.5 C5:0.5', 'C5:0.5 B4:0.5 A4:0.5 G4:0.5', 'G4:0.75 F4:0.25 E4:0.5 E4:0.5',
      'E4:0.5 E4:0.5 E4:0.5 E4:0.25 F4:0.25', 'G4:1.5 F4:0.25 E4:0.25', 'D4:0.5 D4:0.5 D4:0.5 D4:0.25 E4:0.25',
      'F4:1.5 E4:0.25 D4:0.25', 'C4:0.5 C5:1 A4:0.5', 'G4:0.75 F4:0.25 E4:0.5 F4:0.5', 'E4:1 D4:1', 'C4:2',
    ].join(' '),
    chords: 'C C G C F G C C C C C C C G G F C G C',
  },
];
