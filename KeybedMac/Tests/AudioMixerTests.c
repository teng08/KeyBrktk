#include "AudioMixer.h"
#include <assert.h>
#include <math.h>
#include <pthread.h>
#include <stdatomic.h>
#include <stdio.h>

typedef struct {
    KBAudioMixer *mixer;
    int sampleID;
    _Atomic int *finished;
} Producer;

static void *produce(void *context) {
    Producer *producer = context;
    for (int i = 0; i < 20000; ++i) {
        (void)KBMixerPlay(producer->mixer, producer->sampleID, 0.5f);
    }
    atomic_fetch_add(producer->finished, 1);
    return NULL;
}

int main(void) {
    KBAudioMixer *mixer = KBMixerCreate();
    assert(mixer);
    float sample[] = {0.5f, -0.25f, 0.125f, 0};
    int sampleID = KBMixerAddSample(mixer, sample, 4);
    assert(sampleID == 0);
    // The mixer owns its copy of the recording.
    sample[0] = 0;
    float output[128];
    assert(!KBMixerRender(mixer, output, 128));
    assert(!KBMixerPlay(mixer, -1, 1));
    assert(!KBMixerPlay(mixer, 100, 1));
    assert(!KBMixerPlay(mixer, sampleID, NAN));
    KBMixerSetVolume(mixer, 1);
    assert(KBMixerPlay(mixer, sampleID, 1));
    assert(KBMixerRender(mixer, output, 1));
    float singlePeak = output[0];
    assert(singlePeak > 0.4f && singlePeak < 0.5f);
    // Rendering in short blocks retains sample position rather than restarting.
    assert(KBMixerRender(mixer, output, 1));
    assert(output[0] < 0);
    assert(KBMixerRender(mixer, output, 128));
    assert(output[0] > 0 && output[1] == 0 && output[127] == 0);
    assert(!KBMixerRender(mixer, output, 128));
    // Both hits start in the next block and overlap without cutting each other.
    assert(KBMixerPlay(mixer, sampleID, 1));
    assert(KBMixerPlay(mixer, sampleID, 1));
    assert(KBMixerRender(mixer, output, 128));
    assert(output[0] > singlePeak);
    // Bound the queue and output, even during bursts much faster than typing.
    for (int i = 0; i < 255; ++i) assert(KBMixerPlay(mixer, sampleID, 1));
    assert(!KBMixerPlay(mixer, sampleID, 1));
    assert(KBMixerRender(mixer, output, 128));
    for (int i = 0; i < 128; ++i) assert(isfinite(output[i]) && fabsf(output[i]) <= 1);
    assert(KBMixerPlay(mixer, sampleID, 1));
    KBMixerSetVolume(mixer, 0);
    assert(!KBMixerRender(mixer, output, 128));
    for (int i = 0; i < 128; ++i) assert(output[i] == 0);
    KBMixerSetVolume(mixer, 1);
    // UI and event producers can enqueue while the audio consumer renders.
    _Atomic int finished = 0;
    Producer context = {mixer, sampleID, &finished};
    pthread_t first, second;
    assert(pthread_create(&first, NULL, produce, &context) == 0);
    assert(pthread_create(&second, NULL, produce, &context) == 0);
    while (atomic_load(&finished) < 2) {
        KBMixerRender(mixer, output, 128);
        for (int i = 0; i < 128; ++i) assert(isfinite(output[i]) && fabsf(output[i]) <= 1);
    }
    pthread_join(first, NULL);
    pthread_join(second, NULL);
    // Enough banks for the preset library, with a bounded registration limit.
    float extra = 0.125f;
    for (int i = 1; i < 512; ++i) assert(KBMixerAddSample(mixer, &extra, 1) == i);
    assert(KBMixerAddSample(mixer, &extra, 1) == -1);
    KBMixerRender(mixer, output, 128);
    assert(KBMixerPlay(mixer, 511, 1));
    assert(KBMixerRender(mixer, output, 128));
    assert(output[0] > 0.12f && output[1] == 0);
    KBMixerDestroy(mixer);
    puts("PASS: immediate rendering, sample lifetime, overlap, bounded bursts, mute, concurrent producers and 512-sample capacity.");
    return 0;
}
