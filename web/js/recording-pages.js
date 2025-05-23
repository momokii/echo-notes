class TranslatedAudioText {
    constructor(text, chunk_idx) {
        this.translated_text = text
        this.chunk_idx = chunk_idx
    }
}

let BASE_SUMMARIES_URL = '/api/summaries'

// 15 seconds, this for max waiting for wait the last chunk processed after click stop record button
// choose 15 for processed the last chunk just for give enough headroom because on testing, on 1 minute chunk earlier cost like 8 seconds (on bad network)
const MAX_WAITING_TIME = 15000

// chunk duration for recording, 60 seconds for each chunk
const CHUNK_DURATION = 60 * 1000 // on milliseconds 

let mediaRecorder
let audioChunks = []
let recorderArray = []
let isRecording = false
let startTime
let timerInterval
let translatedText = []
let translatedTextFull = ''
let total_recorder = 1

// Cache jQuery selectors
const $recordButton = $('#recordButton')
const $playButton = $('#playButton')
const $downloadButton = $('#downloadButton')
const $audioPlayer = $('#audioPlayer')
const $timerDisplay = $('#timer')
const $recordingIndicator = $('#recordingIndicator')
const $statusText = $('#status-text')

function updateTimer() {
    const now = Date.now();
    const diff = now - startTime;
    const minutes = Math.floor(diff / 60000);
    const seconds = Math.floor((diff % 60000) / 1000);
    $timerDisplay.text(
        `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
    );
}

// because the when recording is chunked, we need to merge all the chunks if needed the full audio
function getFullAudio() {
    return new Blob(audioChunks, {
        type: 'audio/webm'
    })
}

async function startRecording() {
    try {
        // hide summary section
        hideSummary()

        const stream = await navigator.mediaDevices.getUserMedia({
            audio: true
        });
        $statusText.text('Recording...')

        // reset data
        audioChunks = []
        recorderArray = []
        translatedText = []
        translatedTextFull = ''
        total_recorder = 1

        // Create a new recorder for each chunk
        async function createNewRecorder() {
            const mediaRecorder = new MediaRecorder(stream)
            let chunkData = []

            mediaRecorder.ondataavailable = function (event) {
                if (event.data.size > 0) {
                    chunkData.push(event.data)
                }
            };

            mediaRecorder.onstop = async function () {
                // Create a complete WebM file from this chunk's data
                const audioBlob = new Blob(chunkData, {
                    type: 'audio/webm'
                })
                const chunkNumber = audioChunks.length + 1
                audioChunks.push(audioBlob)

                sendChunkToLLM(audioBlob, chunkNumber)

                // Start recording the next chunk
                if (isRecording) {
                    createNewRecorder()
                    total_recorder++
                }
            };

            // Start recording this chunk
            mediaRecorder.start()

            // Stop after desired duration
            setTimeout(() => {
                if (mediaRecorder.state !== 'inactive') {
                    mediaRecorder.stop()
                }
            }, CHUNK_DURATION)

            return mediaRecorder
        }

        // Start the first recorder
        mediaRecorder = await createNewRecorder()
        recorderArray.push(mediaRecorder)
        isRecording = true

        // ui update
        startTime = Date.now()
        timerInterval = setInterval(updateTimer, 1000)
        $recordButton.html('<i class="fas fa-stop">Stop</i>').removeClass('btn-danger').addClass('btn-warning')
        $recordingIndicator.addClass('active')

    } catch (err) {
        $statusText.text('Failed to access microphone')
        showInfoModal('Failed to access microphone. Please ensure you have granted permission.', 'Failed access microphone')
    }
}

async function stopRecording() {
    // Set global recording state to false
    isRecording = false

    // Stop the current recorder (and any others that might be active)
    if (recorderArray && recorderArray.length > 0) {
        recorderArray.forEach(recorder => {
            if (recorder && recorder.state !== 'inactive') {
                recorder.stop()
            }

            // Stop all tracks on each recorder's stream
            if (recorder && recorder.stream) {
                recorder.stream.getTracks().forEach(track => track.stop())
            }
        })
    }

    // Stop the timer
    clearInterval(timerInterval)

    // UI update
    $recordButton.html('<i class="fas fa-microphone">Start New Record</i>').removeClass('btn-warning').addClass('btn-danger')
    $recordingIndicator.removeClass('active')


    // const audioBlob = new Blob(audioChunks, { type: 'audio/webm' })
    // const audioUrl = URL.createObjectURL(audioBlob)

    // // ui and player update
    $statusText.text('Recording finished')
    // $audioPlayer.prop('src', audioUrl)
    // $playButton.prop('disabled', false)
    // $downloadButton.prop('disabled', false)
}


// V1 SIMPLE MODE
async function sendChunkToLLM(audioBlob, chunk_number) {
    try {

        const formData = new FormData();
        formData.append('audio', audioBlob, `chunk-${chunk_number}.webm`);
        formData.append('chunkNumber', chunk_number)

        // send req
        const response = await fetch('/api/audio/chunks', {
            method: 'POST',
            body: formData
        })
        const resp = await response.json()

        if (resp.error) {
            throw new Error(resp.message)
        } else {

            // update translated text
            const data = resp.data
            const translated_text_obj = new TranslatedAudioText(data.translated_text, data.chunk_number - 1)
            translatedText.push(translated_text_obj)
        }


    } catch (e) {
        showInfoModal(e.message, 'Failed to process audio')
    }
}


async function summariesTranslatedAudio() {

    const return_data = {
        simple_summaries: '',
        comprehensive_summaries: ''
    }

    try {
        showLoader("Summarizing Audio...")

        // sort translated text from chunk index
        translatedText.sort((a, b) => a.chunk_idx - b.chunk_idx)

        // get full text
        translatedTextFull = translatedText.map(t => t.translated_text).join(' ')

        const response = await fetch('/api/audio/summaries', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                full_translated_text: translatedTextFull
            })
        })
        const resp = await response.json()

        if (resp.error) {
            throw new Error(resp.message)
        } else {
            const data = resp.data.summaries_data

            // display summary to the user and using ternary operator to check if the data is exist or not (just in case)
            displaySummary(data.tldr_summary || '', data.comprehensive_summary || '')

            return_data.simple_summaries = data.tldr_summary || ''
            return_data.comprehensive_summaries = data.comprehensive_summary || ''
        }

    } catch (e) {
        showInfoModal(e.message, 'Failed to summarize data')
    } finally {
        hideLoader()

        return return_data
    }
}

async function saveSummariesData(simple_summaries, comprehensive_summaries) {
    showLoader("Saving Summaries Data...")

    try {

        const options = {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        }
        const datetime = new Date().toLocaleString('en-US', options)

        // name and description will be the same, just for default value
        const req_body = JSON.stringify({
            name: datetime,
            description: 'Meeting Summary',
            simple_summaries: simple_summaries,
            comprehensive_summaries: comprehensive_summaries
        })

        const response = await fetch(BASE_SUMMARIES_URL, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: req_body
        })
        const resp = await response.json()

        if (resp.error) throw new Error(resp.message)
        else {
            showInfoModal('Summaries data saved successfully!', 'Save Summaries Data')

            return true
        }

    } catch (e) {
        showInfoModal(e.message, 'Failed to save summaries data')

        return false

    } finally {
        hideLoader()
    }
}

// hide summary
function hideSummary() {
    $('#summarySection').addClass('d-none')

    $('#tldrContent').html(`<p></p>`)

    $('#comprehensiveContent').html(`<md-block></md-block>`)

}

// This would be called after your recording is processed and summary is generated
function displaySummary(tldrText, comprehensiveMarkdown) {
    // Show the summary section
    $('#summarySection').removeClass('d-none')

    // Populate TLDR
    $('#tldrContent').html(`<p>${tldrText}</p>`)

    // Render markdown for comprehensive summary
    $('#comprehensiveContent').html(marked.parse(comprehensiveMarkdown))

    // Scroll to summary section
    $('html, body').animate({
        scrollTop: $("#summarySection").offset().top - 20
    }, 500)
}

async function checkReduceToken() {
    showLoader("Checking Token...")
    let isSufficient = false

    try {
        const response = await fetch('/api/audio/summaries/cost', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        const resp = await response.json()

        if (resp.error) throw new Error(resp.message)
        else {
            const data = resp.data

            // update user credit token
            $('#user_credit_token').text(data.new_credit_token)
            isSufficient = true
        }

    } catch (e) {
        showInfoModal(e.message, 'Failed to reduce token')
    } finally {
        hideLoader()
        return isSufficient
    }
}

$('document').ready(async function () {
    hideLoader()

    // Event Handlers
    $recordButton.on('click', async function () {
        if (!isRecording) {
            const isSufficient = await checkReduceToken()
            if (isSufficient) await startRecording()
        } else {
            await stopRecording()

            showLoader("Processed Audio...")

            // before summarize, check if the last chunk is already processed, doing check by comparing the length of total translated text and recorder array
            // and to avoid infinite loop, we set the max waiting time to 15 seconds
            let wait_time = 0

            while ((translatedText.length < total_recorder) && (wait_time < MAX_WAITING_TIME)) {
                wait_time += 1000
                await new Promise(resolve => setTimeout(resolve, 1000))
            }
            hideLoader()

            // process summary
            const summaries_data = await summariesTranslatedAudio()

            // save summaries data
            if (summaries_data.simple_summaries && summaries_data.comprehensive_summaries) {

                // save summaries to db

                // Retry mechanism to save summaries data. 
                // Attempts up to 10 times until successful or retries are exhausted.
                let try_save_summary_to_db = 0
                let is_success = false

                while (!is_success && try_save_summary_to_db < 10) {
                    is_success = await saveSummariesData(summaries_data.simple_summaries, summaries_data.comprehensive_summaries)

                    try_save_summary_to_db++
                }

                if (is_success) {
                    showInfoModal('Summaries data saved successfully!', 'Save Summaries Data')
                } else {
                    showInfoModal('Failed to save summaries data', 'Process & Save Summaries Data')
                }

            } else {
                showInfoModal('Failed to process and save summaries data', 'Process & Save Summaries Data')
            }
        }
    })

    $playButton.on('click', function () {
        $statusText.text('Playing recording...')
        $audioPlayer[0].play()
    })

    $downloadButton.on('click', function () {
        const audioBlob = new Blob(audioChunks, {
            type: 'audio/webm'
        })
        const url = URL.createObjectURL(audioBlob)
        const $link = $('<a>')
            .attr({
                href: url,
                download: 'recording.webm'
            })
            .hide()
            .appendTo('body')

        $link[0].click()
        $link.remove()
        URL.revokeObjectURL(url)
        $statusText.text('Recording downloaded')
    })

    // Event listener for audio player
    $audioPlayer.on('ended', function () {
        $statusText.text('Playback finished')
    })

    // function to copy summary to clipboard
    $('#copySummaryBtn').click(function () {
        const tldr = $('#tldrContent').text()
        const comprehensive = $('#comprehensiveContent').text()
        const fullText = `TLDR:\n${tldr}\n\nComprehensive Summary:\n${comprehensive}`

        navigator.clipboard.writeText(fullText).then(function () {
            showInfoModal('Summary copied to clipboard!', 'Copy Summary')
        }, function () {
            showInfoModal('Failed to copy text!', 'Copy Summary')
        })
    })

    // function to download summary as text file
    $('#downloadSummaryBtn').click(function () {
        const tldr = $('#tldrContent').text()
        const comprehensive = $('#comprehensiveContent').text()
        const fullText = `TLDR:\n${tldr}\n\nComprehensive Summary:\n${comprehensive}`

        const blob = new Blob([fullText], {
            type: 'text/plain'
        })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = 'meeting-summary.txt'
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
    })

})