#include "AudioMixer.h"
#include <math.h>
#include <pthread.h>
#include <stdatomic.h>
#include <stdlib.h>
#include <string.h>

#define KB_MAX_SAMPLES 512
#define KB_MAX_VOICES 32
#define KB_QUEUE_SIZE 256
typedef struct { float *frames; uint32_t count; } KBSample;
typedef struct { int sampleID; float gain; } KBCommand;
typedef struct {
    int sampleID;
    uint32_t position;
    float gain;
    uint64_t age;
    bool active;
} KBVoice;
struct KBAudioMixer {
    KBSample samples[KB_MAX_SAMPLES];
    int sampleCount;
    KBCommand commands[KB_QUEUE_SIZE];
    _Atomic uint32_t readIndex;
    _Atomic uint32_t writeIndex;
    _Atomic float volume;
    // Serializes event/UI producers; the audio consumer never touches this lock.
    pthread_mutex_t producerLock;
    KBVoice voices[KB_MAX_VOICES];
    uint64_t age;
};
KBAudioMixer *KBMixerCreate(void) {
    KBAudioMixer *mixer = calloc(1, sizeof(*mixer));
    if (!mixer) return NULL;
    atomic_init(&mixer->readIndex, 0);
    atomic_init(&mixer->writeIndex, 0);
    atomic_init(&mixer->volume, 0.8f);
    if (pthread_mutex_init(&mixer->producerLock, NULL) != 0) { free(mixer); return NULL; }
    return mixer;
}
void KBMixerDestroy(KBAudioMixer *mixer) {
    if (!mixer) return;
    for (int i = 0; i < mixer->sampleCount; ++i) free(mixer->samples[i].frames);
    pthread_mutex_destroy(&mixer->producerLock);
    free(mixer);
}
int KBMixerAddSample(KBAudioMixer *mixer, const float *frames, uint32_t count) {
    if (!frames || count == 0 || mixer->sampleCount == KB_MAX_SAMPLES) return -1;
    float *copy = malloc(sizeof(float) * count);
    if (!copy) return -1;
    memcpy(copy, frames, sizeof(float) * count);
    int sampleID = mixer->sampleCount++;
    mixer->samples[sampleID] = (KBSample){copy, count};
    return sampleID;
}
bool KBMixerPlay(KBAudioMixer *mixer, int sampleID, float gain) {
    if (sampleID < 0 || sampleID >= mixer->sampleCount || !isfinite(gain)) return false;
    pthread_mutex_lock(&mixer->producerLock);
    uint32_t write = atomic_load_explicit(&mixer->writeIndex, memory_order_relaxed);
    uint32_t next = (write + 1) % KB_QUEUE_SIZE;
    if (next == atomic_load_explicit(&mixer->readIndex, memory_order_acquire)) {
        pthread_mutex_unlock(&mixer->producerLock);
        return false;
    }
    mixer->commands[write] = (KBCommand){sampleID, gain};
    atomic_store_explicit(&mixer->writeIndex, next, memory_order_release);
    pthread_mutex_unlock(&mixer->producerLock);
    return true;
}
void KBMixerSetVolume(KBAudioMixer *mixer, float volume) {
    if (isfinite(volume)) atomic_store_explicit(&mixer->volume, fminf(1, fmaxf(0, volume)), memory_order_relaxed);
}
static float limit(float value) {
    // Smooth saturation prevents overlapping hits from clipping the output.
    if (value >= 3) return 1;
    if (value <= -3) return -1;
    float square = value * value;
    return value * (27 + square) / (27 + 9 * square);
}
bool KBMixerRender(KBAudioMixer *mixer, float *output, uint32_t count) {
    uint32_t read = atomic_load_explicit(&mixer->readIndex, memory_order_relaxed);
    uint32_t write = atomic_load_explicit(&mixer->writeIndex, memory_order_acquire);
    while (read != write) {
        KBCommand command = mixer->commands[read];
        int slot = 0;
        for (int i = 0; i < KB_MAX_VOICES; ++i) {
            if (!mixer->voices[i].active) { slot = i; break; }
            if (mixer->voices[i].age < mixer->voices[slot].age) slot = i;
        }
        mixer->voices[slot] = (KBVoice){command.sampleID, 0, command.gain, ++mixer->age, true};
        read = (read + 1) % KB_QUEUE_SIZE;
    }
    atomic_store_explicit(&mixer->readIndex, read, memory_order_release);
    memset(output, 0, sizeof(float) * count);
    for (int i = 0; i < KB_MAX_VOICES; ++i) {
        KBVoice *voice = &mixer->voices[i];
        if (!voice->active) continue;
        KBSample sample = mixer->samples[voice->sampleID];
        uint32_t remaining = sample.count - voice->position;
        uint32_t frames = remaining < count ? remaining : count;
        for (uint32_t frame = 0; frame < frames; ++frame) {
            output[frame] += sample.frames[voice->position + frame] * voice->gain;
        }
        voice->position += frames;
        voice->active = voice->position < sample.count;
    }
    float volume = atomic_load_explicit(&mixer->volume, memory_order_relaxed);
    bool audible = false;
    for (uint32_t frame = 0; frame < count; ++frame) {
        output[frame] = limit(output[frame] * volume);
        audible |= output[frame] != 0;
    }
    return audible;
}
