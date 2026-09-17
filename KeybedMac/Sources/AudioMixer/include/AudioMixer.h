#ifndef KEYBED_AUDIO_MIXER_H
#define KEYBED_AUDIO_MIXER_H
#include <stdbool.h>
#include <stdint.h>
typedef struct KBAudioMixer KBAudioMixer;
KBAudioMixer *KBMixerCreate(void);
void KBMixerDestroy(KBAudioMixer *mixer);
// Register immutable PCM before starting the engine.
int KBMixerAddSample(KBAudioMixer *mixer, const float *frames, uint32_t count);
bool KBMixerPlay(KBAudioMixer *mixer, int sampleID, float gain);
void KBMixerSetVolume(KBAudioMixer *mixer, float volume);
// Only the audio thread calls Render. It never allocates or takes a lock.
bool KBMixerRender(KBAudioMixer *mixer, float *output, uint32_t count);
#endif
