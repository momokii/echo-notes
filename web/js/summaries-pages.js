let BASE_SUMMARIES_URL = '/api/summaries'
let SEARCH_TIMEOUT = null
let SEARCH_SUMMARIES = ''
let FILTER_SUMMARIES = 'newest'
let PAGE = 1
let PER_PAGE = 5
let PAGE_GROUPING = 1
let PER_PAGE_GROUPING = 5

const MAX_SELECTED_GROUP_SUMMARIES = 5

async function changePage(page, per_page) {
    PAGE = page
    PER_PAGE = per_page
    loadSummaries()
}

async function updatePagination(pagination) {
    // structure of pagination object
    // pagination = {
    //     current_page: 1,
    //     total_page: 1,
    //     total_data: 1,
    //     per_page: 1
    // } 

    const paginationContainer = $('.pagination')
    paginationContainer.empty()

    const {
        current_page,
        total_page,
        total_data,
        per_page
    } = pagination
    const rangePageShow = 1

    // prviouse buttom
    paginationContainer.append(`
            <li class="page-item ${current_page === 1 ? 'disabled' : ''}">
                <button class="page-link" onclick="changePage(${current_page - 1}, ${PER_PAGE})" ${pagination.current_page === 1 ? 'disabled' : ''}>Previous</button>
            </li>
        `)

    // Tambahkan halaman pertama
    if (current_page > rangePageShow + 1) {
        paginationContainer.append(`
                <li class="page-item">
                    <button class="page-link" onclick="changePage(1, ${PER_PAGE})">1</button>
                </li>
                <li class="page-item disabled">
                    <span class="page-link">...</span>
                </li>
            `);
    }

    // Tambahkan halaman di sekitar halaman aktif
    const start = Math.max(1, current_page - rangePageShow);
    const end = Math.min(total_page, current_page + rangePageShow);

    for (let i = start; i <= end; i++) {
        paginationContainer.append(`
                <li class="page-item ${current_page === i ? 'active' : ''}">
                    <button class="page-link" onclick="changePage(${i},  ${PER_PAGE})">${i}</button>
                </li>
            `);
    }

    // Tambahkan halaman terakhir
    if (current_page < total_page - rangePageShow) {
        paginationContainer.append(`
                <li class="page-item disabled">
                    <span class="page-link">...</span>
                </li>
                <li class="page-item">
                    <button class="page-link" onclick="changePage(${total_page}, ${PER_PAGE})">${total_page}</button>
                </li>
            `);
    }

    // next button
    paginationContainer.append(`
            <li class="page-item ${current_page === total_page? 'disabled' : ''}">
                <button class="page-link" onclick="changePage(${current_page + 1}, ${PER_PAGE})" ${current_page === total_page ? 'disabled' : ''}>Next</button>
            </li>
        `)

}


// load summaries data on dashboard (because using url on parameter, thos can use for summaries user and grouping summaries data) 
async function loadSummariesAPI(url) {

    try {
        const resp = await fetch(url, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            },
        })
        const response = await resp.json()

        if (response.error) throw new Error(response.message)
        else return {
            isError: false,
            data: response.data,
            message: response.message
        }

    } catch (e) {
        return {
            isError: true,
            data: null,
            message: e.message
        }
    }
}


// ================== GROUPING SUMMARIES RELATED FUNCTION

// Variables for infinite scroll
let isLoadingMore = false
let hasMoreData = true

// Check selected summary list and enable/disable the summarize button accordingly (for add group summaries)
function isThereSelectedGroupData() {
    const selectedList = $('#selectedSummary');

    const noSelectionsMessage = selectedList.find('.text-muted');
    
    if (selectedList.children().length === 0 || 
        (selectedList.children().length === 1 && selectedList.children().first().is('p.text-muted'))) {
        $('#confirmaddGroupBtn').prop('disabled', true);
        
        // Only add the message if it doesn't already exist
        if (noSelectionsMessage.length === 0) {
            selectedList.append(`<p class="text-muted">No Summaries Selected</p>`);
        }
    } else {
        $('#confirmaddGroupBtn').prop('disabled', false);
        
        // Remove the "No Summaries Selected" message if items are added
        noSelectionsMessage.remove();
    }
}


// for load summaries data on group selection on group summaries function (with infinite scroll)
async function loadSummariesGroupingSelection(append = false) {
    if (isLoadingMore) return;

    isLoadingMore = true;
    SEARCH_SUMMARIES = $('#groupSummariesSearch').val() || '';
    FILTER_SUMMARIES = 'newest';

    // Add a small loading indicator at the bottom of the list when loading more
    if (append) {
        $('#groupSummariesList').append('<li class="list-group-item text-center" id="loading-indicator"><div class="spinner-border spinner-border-sm text-primary" role="status"></div> Loading more...</li>');
    } else {
        showLoader();
    }

    let LOAD_DATA_BASE_URL = BASE_SUMMARIES_URL + `?search=${SEARCH_SUMMARIES}&order_by=${FILTER_SUMMARIES}&page=${PAGE_GROUPING}&per_page=${PER_PAGE_GROUPING}`;

    try {
        let {
            isError,
            data,
            message
        } = await loadSummariesAPI(LOAD_DATA_BASE_URL);

        if (isError) throw new Error(message);
        else {
            const summariesGroupList = $('#groupSummariesList');

            // Remove loading indicator if appending
            if (append) {
                $('#loading-indicator').remove();
            } else {
                summariesGroupList.empty();
            }

            const summariesData = data.summaries;
            const paginationData = data.pagination;

            // Check if there's more data to load
            hasMoreData = paginationData.current_page < paginationData.total_page;

            if (summariesData.length === 0 && !append) {
                summariesGroupList.append(`<p class="text-muted">No Summaries Recorded Found</p>`);
            } else if (summariesData.length === 0 && append) {
                // If appending but no new data
                summariesGroupList.append(`<li class="list-group-item text-center">No more data to load</li>`);
                setTimeout(() => {
                    $('.list-group-item:contains("No more data to load")').fadeOut();
                }, 2000);
            } else {
                summariesData.forEach(summary => {
                    // Check if this summary is already in the list or selected list
                    if ($(`#summary-list-${summary.id}`).length === 0 && $(`#selected-summary-${summary.id}`).length === 0) {
                        const description = summary.description.replace(/<br>/g, ' ');
                        const truncatedDesc = description.length > 60 ?
                            description.substring(0, 60) + '...' : description;

                        const summaryItem = `
                                <li class="list-group-item d-flex justify-content-between align-items-center" id="summary-list-${summary.id}">
                                    <div>
                                        <strong>${summary.name}</strong>
                                        <p class="mb-0 text-muted small">${truncatedDesc}</p>
                                    </div>
                                    <button class="btn btn-sm btn-success add-summary-btn" 
                                        data-id="${summary.id}" 
                                        data-name="${summary.name}" 
                                        data-description="${description}">
                                        <i class="fas fa-plus"></i> Add
                                    </button>
                                </li>
                            `;
                        summariesGroupList.append(summaryItem);
                    }
                });

                // Add event listeners for the add buttons
                $('.add-summary-btn').off('click').on('click', function () {
                    const id = $(this).data('id');
                    const name = $(this).data('name');
                    const description = $(this).data('description');

                    // Add to selected list and check if it was successful
                    const isSuccess = addToSelectedSummaries(id, name, description)

                    // Remove from available list just if success add new data 
                    if (isSuccess) $(`#summary-list-${id}`).remove()
                });
            }

            isThereSelectedGroupData();

            // If successfully appended, increment page for next load
            if (append && summariesData.length > 0) {
                PAGE_GROUPING++;
            }
        }
    } catch (e) {
        if (!append) {
            showInfoModal('Failed to Load Summaries: ' + e.message, 'Failed to load summaries');
        } else {
            $('#loading-indicator').html('Error loading more data. Try again.');
            setTimeout(() => {
                $('#loading-indicator').remove();
            }, 3000);
        }
    } finally {
        isLoadingMore = false;
        if (!append) {
            hideLoader();
        }
    }
}

// Function to add a summary to the selected list
function addToSelectedSummaries(id, name, description) {
    const selectedList = $('#selectedSummary')

    // here to add condition if the selected items is max so user cant add new items here
    if (selectedList.children().length >= MAX_SELECTED_GROUP_SUMMARIES) {
        showInfoModal('You can only select up to 5 summaries for group summarization', 'Max Summaries Reached');

        return false
    }

    const truncatedDesc = description.length > 60 ?
        description.substring(0, 60) + '...' : description

    const selectedItem = `
            <li class="list-group-item d-flex justify-content-between align-items-center" id="selected-summary-${id}">
                <div>
                    <strong>${name}</strong>
                    <p class="mb-0 text-muted small">${truncatedDesc}</p>
                </div>
                <button class="btn btn-sm btn-danger remove-summary-btn" 
                    data-id="${id}" 
                    data-name="${name}" 
                    data-description="${description}">
                    Delete
                </button>
            </li>
        `

    selectedList.append(selectedItem)

    isThereSelectedGroupData()

    // Add event listener for the remove button
    $(`#selected-summary-${id} .remove-summary-btn`).off('click').on('click', function () {
        const id = $(this).data('id')
        const name = $(this).data('name')
        const description = $(this).data('description')

        // Remove from selected list
        $(`#selected-summary-${id}`).remove()

        // check selected summary list and if the length is 0 so disable the summarize button
        isThereSelectedGroupData()

        // Add back to available list if not already present
        if ($(`#summary-list-${id}`).length === 0) {
            const truncatedDesc = description.length > 60 ?
                description.substring(0, 60) + '...' : description

            const summaryItem = `
                    <li class="list-group-item d-flex justify-content-between align-items-center" id="summary-list-${id}">
                        <div>
                            <strong>${name}</strong>
                            <p class="mb-0 text-muted small">${truncatedDesc}</p>
                        </div>
                        <button class="btn btn-sm btn-success add-summary-btn" 
                            data-id="${id}" 
                            data-name="${name}" 
                            data-description="${description}">
                            <i class="fas fa-plus"></i> Add
                        </button>
                    </li>
                `

            $('#groupSummariesList').append(summaryItem)

            // Reattach click event for the new add button
            $(`#summary-list-${id} .add-summary-btn`).off('click').on('click', function () {
                addToSelectedSummaries(id, name, description);
                $(`#summary-list-${id}`).remove()
            })
        }
    })


    return true
}

// Helper function to handle infinite scroll (use for my grouping data and add group summaries modal)
function handleInfiniteScroll(isMyGroupingData = false) {
    // Check if the modal is open and the scroll event is triggered
    // this for flexible modal body
    let modalBody = $('#addGroupSummariesModal .modal-body')
    
    if (isMyGroupingData) modalBody = $('#GroupSummariesDataModal .modal-body')

    modalBody.scroll(function () {
        // Check if scrolled to bottom (with a small threshold)
        if (modalBody.scrollTop() + modalBody.innerHeight() >= modalBody[0].scrollHeight - 20) {
            // Load more data if not already loading and more data exists
            if (!isLoadingMore && hasMoreData) {

                // load and call the function to load more data based on the modal type
                if(!isMyGroupingData) loadSummariesGroupingSelection(true)
                else loadMyGroupSummaries(true)

            }
        }
    });
}

// Reset grouping pagination when opening the modal (use for my grouping data and add group summaries modal)
function resetGroupingSummariesPagination() {
    PAGE_GROUPING = 1
    hasMoreData = true
    
    // below is using for add group summaries modal
    $('#selectedSummary').empty() // clear selected summaries (for add group summaries)
    isThereSelectedGroupData() // if not cann summarize
}

// ======= (ADD GROUP SUMMARIES) GROUPING SUMMARIES FUNCTION =======
// Helper function to collect selected summaries IDs for group summarization
function getSelectedSummariesIds() {
    const selectedIds = []
    $('#selectedSummary li').each(function () {
        const id = $(this).attr('id').replace('selected-summary-', '')
        selectedIds.push(id)
    })
    return selectedIds
}

// grouping summarize function and call to LLM APi
async function summarizeGroupLLM() {
    // get opened modal (the selected grouping data) and will be closed it after the request complete
    const modalElement = $('#addGroupSummariesModal')
    const modalAddGroupData = bootstrap.Modal.getInstance(modalElement)

    const resultSummariesModal = new bootstrap.Modal($('#resultGroupSummariesModal'))

    let selectedId = getSelectedSummariesIds()
    // convert selected id to int
    selectedId = selectedId.map(Number)

    showLoader("Summarizing group data...")

    try {

        const resp = await fetch(BASE_SUMMARIES_URL + '/groups/llm', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                summaries_id: selectedId
            })
        })
        const response = await resp.json()

        // result data
        const overview = response.data.group_summaries_data.overview
        const summaries_data = response.data.group_summaries_data.meeting_summaries
        const next_steps = response.data.group_summaries_data.next_steps

        $('#resultOverviewSummariesGroupData').html(marked.parse(overview))
        $('#resultMeetingSummariesGroupData').html(marked.parse(summaries_data))
        $('#resultNextStepSummariesGroupData').html(marked.parse(next_steps))

        resultSummariesModal.show()
        // Set up an event handler to clear content when modal is hidden
        $('#resultGroupSummariesModal').on('hidden.bs.modal', function () {
            $('#resultOverviewSummariesGroupData').html('')
            $('#resultMeetingSummariesGroupData').html('')
            $('#resultNextStepSummariesGroupData').html('')
        })

        // add new event handler for summarize button
        // save the result to account if user needeed it
        $('#saveGroupSummariesResultBtn').off('click').on('click', async function () {
            await saveGroupSummarizeResult(overview, summaries_data, next_steps);
        });

        resultSummariesModal.show()

    } catch (e) {
        showInfoModal('Failed to Summarize Group: ' + e.message, 'Failed to summarize group')

    } finally {
        hideLoader()
        // Reset the selected summaries after summarization
        $('#selectedSummary').empty()
        isThereSelectedGroupData()

        // hide the modal selected data
        modalAddGroupData.hide()
    }
}

// check if token is sufficient for using group summarize feature
async function checkReduceToken() {
    showLoader("Checking Token...")
    let isSufficient = false

    try {
        const response = await fetch('/api/summaries/groups/cost', {
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
            $('#userCredit').text(data.new_credit_token)
            isSufficient = true
        }

    } catch (e) {
        showInfoModal(e.message, 'Failed to reduce token')
    } finally {
        hideLoader()
        return isSufficient
    }
}

// save the result to account if user needed it
async function saveGroupSummarizeResult(overview, summaries_data, next_steps) {
    // get opened modal and will be closed it if the request is success
    const modalElement = $('#resultGroupSummariesModal')
    const modalResult = bootstrap.Modal.getInstance(modalElement)

    showLoader('Saving group summaries result...')

    try {
        
        const body_req = JSON.stringify({
            overview: overview,
            meeting_summaries: summaries_data,
            next_steps: next_steps
        })

        const response = await fetch(BASE_SUMMARIES_URL + '/groups', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: body_req
        })
        const resp = await response.json()

        if (resp.error) throw new Error(resp.message)
        
        // hide the result modal if success save the summary grouping to avoid user save the same data twice
        modalResult.hide()
        showInfoModal('Saving group summaries result success!', 'Saving group summaries result')

    } catch(e){
        showInfoModal('Failed to save group summaries result: ' + e.message, 'Failed to save group summaries result')
    } finally {
        hideLoader()
    }
}

// ======= (MY GROUP SUMMARIES) GROUPING SUMMARIES FUNCTION =======
// open and show modal for group summaries data list that user created
async function loadMyGroupSummaries(append = false) {
    if(isLoadingMore) return

    isLoadingMore = true
    SEARCH_GROUP_SUMMARIES = $('#groupSummariesDataSearch').val() || ''
    FILTER = 'newest' // always newest for group summaries data because using scroll

    // Add a small loading indicator at the bottom of the list when loading more
    if (append) {
        $('#groupSummariesDataList').append('<li class="list-group-item text-center" id="loading-indicator-group-data"><div class="spinner-border spinner-border-sm text-primary" role="status"></div> Loading more...</li>')
    } else {
        showLoader()
    }

    let LOAD_DATA_BASE_URL = BASE_SUMMARIES_URL + '/groups' + `?search=${SEARCH_GROUP_SUMMARIES}&order_by=${FILTER}&page=${PAGE_GROUPING}&per_page=${PER_PAGE_GROUPING}`

    try {
        let {
            isError,
            data,
            message
        } = await loadSummariesAPI(LOAD_DATA_BASE_URL)

        if (isError) throw new Error(message)
        else {
            
            // load summaries data on group summaries modal
            const summariesGroupList = $('#groupSummariesDataList')
            
            // Remove loading indicator if appending
            if (append) {
                $('#loading-indicator-group-data').remove()
            } else {
                summariesGroupList.empty()
            }

            const summariesGroupData = data.group_summaries || []
            const paginationData = data.pagination

            // Check if there's more data to load
            hasMoreData = paginationData.current_page < paginationData.total_page
            
            if (summariesGroupData.length === 0 && !append) {
                summariesGroupList.append(`<p class="text-muted">No Group Summaries Found</p>`)
            } else if (summariesGroupData.length === 0 && append) {
                // If appending but no new data
                summariesGroupList.append(`<li class="list-group-item text-center">No more data to load</li>`)
                setTimeout(() => {
                    $('.list-group-item:contains("No more data to load")').fadeOut(() => {
                        $(this).remove()
                    })
                }, 2000)
            } else {
                
                summariesGroupData.forEach(summary => {
                    // Check if this summary is already in the list -> to make sure the summary is not duplicated
                    if ($(`#summary-group-list-${summary.id}`).length === 0) {
                        const createDate = new Date(summary.created_at).toLocaleDateString('en-US', {
                            year: 'numeric',
                            month: 'long',
                            day: 'numeric'
                        })
                        
                        // Store summary data safely
                        const summaryData = {
                            id: summary.id,
                            name: summary.name || `Group Summary ${summary.id}`,
                            description: summary.description || '',
                            overview: summary.overview || '',
                            meeting_summaries: summary.meeting_summaries || '',
                            next_steps: summary.next_steps || '',
                            created_at: createDate
                        }
                        
                        // Create a data attribute string with just the ID
                        const summaryItem = `
                            <li class="list-group-item" id="summary-group-list-${summaryData.id}">
                                <div class="d-flex justify-content-between align-items-center">
                                    <div>
                                        <strong>${summaryData.name}</strong>
                                        <p class="mb-0 text-muted small">${summaryData.description || 'No description available'}</p>
                                        <p class="mb-0 text-muted small">Created: ${summaryData.created_at}</p>
                                    </div>
                                    <div>
                                        <button class="btn btn-sm btn-success detail-group-summary-btn" data-id="${summaryData.id}">
                                            Detail
                                        </button>
                                        <button class="btn btn-sm btn-warning edit-group-summary-btn" data-id="${summaryData.id}">
                                            Edit
                                        </button>
                                        <button class="btn btn-sm btn-danger delete-group-summary-btn" data-id="${summaryData.id}">
                                            Delete
                                        </button>
                                    </div>
                                </div>
                            </li>
                        `
                        summariesGroupList.append(summaryItem)
                        
                        // Store the full data in a JavaScript object instead of data attributes
                        // This avoids HTML encoding issues with complex text
                        $(`#summary-group-list-${summaryData.id}`).data('summary', summaryData)
                    }
                })
                
                // Add event listeners for the detail buttons - attached after appending to DOM
                $('.detail-group-summary-btn').off('click').on('click', function() {
                    const summaryId = $(this).data('id')
                    const summaryData = $(`#summary-group-list-${summaryId}`).data('summary')
                    
                    if (!summaryData) {
                        showInfoModal('Could not retrieve summary data', 'Error')
                        return
                    }
                    
                    // Close the group list modal before opening the detail modal
                    const groupListModal = bootstrap.Modal.getInstance(document.getElementById('GroupSummariesDataModal'));
                    if (groupListModal) {
                        groupListModal.hide();
                    }
                    
                    // Small delay to ensure modal is fully hidden
                    setTimeout(() => {
                        // Now open the modal with this data
                        openGroupDetailModal(summaryData);
                    }, 300);
                })
                
                // Add event listeners for delete buttons
                $('.delete-group-summary-btn').off('click').on('click', function() {
                    const summaryId = $(this).data('id')
                    const summaryData = $(`#summary-group-list-${summaryId}`).data('summary')
                    
                    if (!summaryData) {
                        showInfoModal('Could not retrieve summary data', 'Error')
                        return
                    }
                    
                    deleteGroupSummary(summaryId, summaryData.name, summaryData.description)
                })

                // Add event listeners for edit buttons
                $('.edit-group-summary-btn').off('click').on('click', function() {
                    const summaryId = $(this).data('id')
                    const summaryData = $(`#summary-group-list-${summaryId}`).data('summary')
                    
                    if (!summaryData) {
                        showInfoModal('Could not retrieve summary data', 'Error')
                        return
                    }
                    
                    editGroupSummary(summaryId, summaryData.name, summaryData.description)
                })
            }

            // If successfully appended, increment page for next load
            if (append && summariesGroupData.length > 0) {
                PAGE_GROUPING++
            }
        }
    } catch (e) {
        if (!append) {
            showInfoModal('Failed to Load Group Summaries: ' + e.message, 'Failed to load group summaries')
        } else {
            $('#loading-indicator-group-data').html('Error loading more data. Try again.')
            setTimeout(() => {
                $('#loading-indicator-group-data').remove()
            }, 3000)
        }
    } finally {
        isLoadingMore = false
        if (!append) {
            hideLoader()
        }
    }
}

// New function to open the group detail modal
function openGroupDetailModal(summaryData) {
    // Close any open modals first
    const openModals = document.querySelectorAll('.modal.show');
    openModals.forEach(modal => {
        const modalInstance = bootstrap.Modal.getInstance(modal);
        if (modalInstance) {
            modalInstance.hide();
        }
    });
    
    // Get the modal instance
    const detailModal = new bootstrap.Modal(document.getElementById('detailGroupSummariesModal'));
    
    // Set the modal content
    $('#groupSummariesNameDetailModal').text(summaryData.name || '')
    $('#groupSummariesDescriptionDetailModal').text(summaryData.description || '')
    
    // Use try-catch blocks to handle potential errors when parsing markdown for make sure that modal can be opened
    try {
        $('#overviewGroupContent').html(marked.parse(summaryData.overview || ''))
    } catch (e) {
        $('#overviewGroupContent').html('<p class="text-danger">Error parsing overview content</p>')
    }
    
    try {
        $('#meetingSummariesGroupContent').html(marked.parse(summaryData.meeting_summaries || ''))
    } catch (e) {
        $('#meetingSummariesGroupContent').html('<p class="text-danger">Error parsing meeting summaries content</p>')
    }
    
    try {
        $('#nextStepGroupContent').html(marked.parse(summaryData.next_steps || ''))
    } catch (e) {
        $('#nextStepGroupContent').html('<p class="text-danger">Error parsing next steps content</p>')
    }

    // Set up an event handler to clear content when modal is hidden
    $('#detailGroupSummariesModal').on('hidden.bs.modal', function () {
        $('#overviewGroupContent').html('')
        $('#meetingSummariesGroupContent').html('')
        $('#nextStepGroupContent').html('')
    })
    
    // Show the modal
    detailModal.show()
}

// Function to delete a group summary
async function deleteGroupSummary(id, name, description) {
    // First, explicitly close the detail modal if it's open
    try {
        // Get all open modals and close them first
        const openModals = document.querySelectorAll('.modal.show');
        openModals.forEach(modal => {
            const modalInstance = bootstrap.Modal.getInstance(modal);
            if (modalInstance) {
                modalInstance.hide();
            }
        });
        
        // Small delay to ensure modal is fully hidden before showing the new one
        await new Promise(resolve => setTimeout(resolve, 300));
    } catch (error) {
        showInfoModal('Failed to close modal: ' + error.message, 'Error')
    }
    
    // Now show the delete confirmation modal
    const deleteModal = new bootstrap.Modal($('#deleteSummariesModal'))
    $('#summariesNameDeleteModal').text(name)
    $('#summariesDescriptionDeleteModal').text(description)
    deleteModal.show()

    $('#confirmDeleteBtn').one('click', async function () {

        showLoader('Deleting group summary...')
    
        try {
            const resp = await fetch(`${BASE_SUMMARIES_URL}/groups/${id}`, {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json'
                }
            })
            
            const response = await resp.json()
            if (response.error) throw new Error(response.message)
            
            // Remove the item from the DOM
            $(`#summary-group-list-${id}`).remove()

            showInfoModal(`Successfully deleted group summary "${name}"`, 'Delete Successful')
            
        } catch (e) {
            showInfoModal('Failed to delete group summary: ' + e.message, 'Delete Failed')
        } finally {
            deleteModal.hide()
            hideLoader()
        }
        
    })
}

// Function to edit a grup summary 
async function editGroupSummary(id, name, description) {
    // First, explicitly close the detail modal if it's open
    try {
        // Get all open modals and close them first
        const openModals = document.querySelectorAll('.modal.show');
        openModals.forEach(modal => {
            const modalInstance = bootstrap.Modal.getInstance(modal);
            if (modalInstance) {
                modalInstance.hide();
            }
        })
        
        // Small delay to ensure modal is fully hidden before showing the new one
        await new Promise(resolve => setTimeout(resolve, 300));
    } catch (error) {
        showInfoModal('Failed to close modal: ' + error.message, 'Error')
    }

    // show the edit modal
    const editModal = new bootstrap.Modal($('#editSummariesModal'))
    $('#summariesNameEdit').val(name)
    $('#summariesDescriptionEdit').val(description)
    editModal.show()

    $('#editSummariesBtn').off('click').click(async function () {
        event.preventDefault()

        const new_name = $('#summariesNameEdit').val()
        const new_description = $('#summariesDescriptionEdit').val().replace(/\n/g, '<br>')

        showLoader("Editing group summary data...")
        try {

            const resp = await fetch(BASE_SUMMARIES_URL + '/groups/' + id, {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    name: new_name,
                    description: new_description
                })
            })
            
            const response = await resp.json()

            if(response.error) throw new Error(response.message)

            showInfoModal(`Edit Summaries Group Data Success!`, 'Success Edit Data')

        } catch(e) {
            showInfoModal('Failed to edit group summary: ' + e.message, 'Edit Failed')
        } finally {
            hideLoader()
            editModal.hide()
        }
    })
}


// ================== SUMMARIES DATA ON DASHBOARD

// edit modal function and request to server
async function openEditModal(id, name, description) {
    const editModal = new bootstrap.Modal($('#editSummariesModal'))

    $('#summariesNameEdit').val(name)
    $('#summariesDescriptionEdit').val(description)
    editModal.show()

    // edit room modal 
    $('#editSummariesBtn').off('click').click(async function () {
        event.preventDefault()

        const new_name = $('#summariesNameEdit').val()
        const new_description = $('#summariesDescriptionEdit').val().replace(/\n/g, '<br>');

        const dataReq = JSON.stringify({
            name: new_name,
            description: new_description
        })
        editModal.hide()
        showLoader()

        try {
            const URL_EDIT = BASE_SUMMARIES_URL + `/${id}`

            const resp = await fetch(URL_EDIT, {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: dataReq
            })
            const response = await resp.json()

            if (response.error) throw new Error('Failed to edit summaries data' + ': ' + response.message)
            else {
                await loadSummaries()
                showInfoModal(`Success edit summaries data`, 'Edit Summaries Data Success')
            }

        } catch (e) {
            showInfoModal(e.message, 'Edit Room Failed')
        } finally {
            hideLoader()
        }
    })
}

// delete modal function and request to server
async function openDeleteModal(id, name, description) {
    const deleteModal = new bootstrap.Modal($('#deleteSummariesModal'))
    $('#summariesNameDeleteModal').text(name)
    $('#summariesDescriptionDeleteModal').text(description)
    deleteModal.show()

    $('#confirmDeleteBtn').one('click', async function () {
        event.preventDefault()

        deleteModal.hide()
        showLoader()

        try {
            const URL_DELETE = BASE_SUMMARIES_URL + `/${id}`

            const resp = await fetch(URL_DELETE, {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json'
                }
            })
            const response = await resp.json()

            if (response.error) throw new Error('Failed to delete summary data with name ' + name + ': ' + response.message)
            else {
                hideLoader()

                await loadSummaries()

                showInfoModal(`Success delete summary data with name ${name}`, 'Delete Room Success')
            }

        } catch (e) {
            showInfoModal(e.message, 'Delete Summary Data Failed')
            hideLoader()
        }
    })
}

// open modal detail summary
async function openDetailModal(id, name, description, simple_summary, comprehensive_summary) {
    const detailModal = new bootstrap.Modal($('#detailSummariesModal'))
    $('#summariesNameDetailModal').text(name)
    $('#summariesDescriptionDetailModal').text(description)
    detailModal.show()

    // Populate TLDR
    $('#tldrContent').html(`<p>${simple_summary}</p>`)

    // Render markdown for comprehensive summary
    $('#comprehensiveContent').html(marked.parse(comprehensive_summary))

}

// for load summaries data on dashboard
async function loadSummaries() {
    let LOAD_DATA_BASE_URL = BASE_SUMMARIES_URL + `?search=${SEARCH_SUMMARIES}&order_by=${FILTER_SUMMARIES}&page=${PAGE}&per_page=${PER_PAGE}`

    showLoader()

    try {
        let {
            isError,
            data,
            message
        } = await loadSummariesAPI(LOAD_DATA_BASE_URL)

        if (isError) throw new Error(message)
        else {

            const summariesList = $('#roomList')
            summariesList.empty()

            const summariesData = data.summaries
            const paginationData = data.pagination

            if (summariesData.length === 0) summariesList.append(`<p class="text-muted">No Summaries Recorded Found</p>`)
            else {

                summariesData.forEach(summary => {

                    const create_date = new Date(summary.created_at).toLocaleDateString('en-US', {
                        year: 'numeric',
                        month: 'long',
                        day: 'numeric'
                    })
                    const description = summary.description.replace(/<br>/g, '\n')

                    let comprehensiveEscaped = JSON.stringify(summary.comprehensive_summaries).replace(/"/g, '&quot;');

                    let summaryCard = `
                            <div class="room-card text-justify" id="summary-${summary.id}">
                                <div class="card-body">
                                    <h5 class="fw-bold text-primary mb-2">${summary.name}</h5>
                                    <p class="text-muted mb-3">
                                        <strong>Description:</strong><br>
                                        <span style="text-align: justify;">${description}</span>
                                    </p>
                                    <p class="text-muted mb-1">
                                        <strong>Created At:</strong> 
                                        <span class="fw-bold">${create_date}</span>
                                    </p>
                                </div>
                                <div class="card-footer room-actions">
                                    <button class="btn btn-sm btn-success" onclick="openDetailModal(${summary.id}, '${summary.name}', '${description}', '${summary.simple_summaries}', ${comprehensiveEscaped})">Detail</button>

                                    <button class="btn btn-sm btn-warning" onclick="openEditModal(${summary.id}, '${summary.name}', '${description}')">Edit</button>

                                    <button class="btn btn-sm btn-danger" onclick="openDeleteModal(${summary.id},'${summary.name}', '${description}')">Delete</button>
                                </div>
                            </div>
                        `

                    summariesList.append(summaryCard)

                })

            }
            updatePagination(paginationData)
        }

    } catch (e) {
        showInfoModal('Failed to Load Summaries: ' + e.message, 'Failed to load summaries')
    } finally {
        hideLoader()
    }

}

$(document).ready(async function () {

    // ============= INITIATE PAGE 
    hideLoader()

    loadSummaries()

    // ============= SUMMARIES GROUP FEATURE
    $('#addGroupSummaries').click(async function () {
        resetGroupingSummariesPagination()
        const addGroupModal = new bootstrap.Modal($('#addGroupSummariesModal'))
        addGroupModal.show()

        // Initialize infinite scroll after modal is shown
        $('#addGroupSummariesModal').on('shown.bs.modal', function () {
            handleInfiniteScroll()
        })

        // Load first batch of data
        loadSummariesGroupingSelection()
    })

    $('#myGroupSummaries').click(async function () {
        resetGroupingSummariesPagination() // using for reset pagination and hasMoreData var
        const groupSummariesModal = new bootstrap.Modal($('#GroupSummariesDataModal'))
        groupSummariesModal.show()

        // Initialize infinite scroll after modal is shown
        $('#GroupSummariesDataModal').on('shown.bs.modal', function () {
            handleInfiniteScroll(true) // using true for my grouping data
        })

        loadMyGroupSummaries()
    })

    // search functionality for group summaries (ADD TO USE SUMMARIES GROUP)
    $('#groupSummariesSearch').on('input', function () {
        clearTimeout(SEARCH_TIMEOUT)

        SEARCH_TIMEOUT = setTimeout(function () {
            // Reset pagination when searching
            PAGE_GROUPING = 1
            hasMoreData = true
            loadSummariesGroupingSelection()
        }, 500)
    })

    // send request to summarize grouping data
    $('#confirmaddGroupBtn').click(async function () {

        // check if the token user have is sufficient or not
        const isSufficient = await checkReduceToken()
        if (!isSufficient) {
            showInfoModal('You do not have sufficient token to use this feature', 'Token Insufficient')
            return
        }

        // if usfficient so can use the feature
        await summarizeGroupLLM()
    })

    // search functionality for group summaries (GROUP SUMMARIES DATA)
    $('#groupSummariesDataSearch').on('input', function () {
        clearTimeout(SEARCH_TIMEOUT)

        SEARCH_TIMEOUT = setTimeout(function () {

            // Reset pagination when searching
            PAGE_GROUPING = 1
            hasMoreData = true
            loadMyGroupSummaries()

        }, 1000)
    })


    // ============== COPY AND DOWNLOAD BUTTON FOR SINGLE AND GROUPING SUMMARIES RESULT

    // function to get full text grouping summaries for copy and download
    function getFullTextGroupingSummaries() {
        let isNotNull = true

        // below is for grouping summary result
        let overviewGroupingSummaries = $('#resultOverviewSummariesGroupData').text()
        let meetingGroupingSummaries = $('#resultMeetingSummariesGroupData').text()
        let nextStepGroupingSummaries = $('#resultNextStepSummariesGroupData').text()

        // if all grouping summaries is empty, return empty string
        if (!overviewGroupingSummaries && !meetingGroupingSummaries && !nextStepGroupingSummaries) {
            isNotNull = false
        }

        if (!isNotNull) {
            // below for grouping summarys data (if user click the detail button)
            overviewGroupingSummaries = overviewGroupingSummaries || $('#overviewGroupContent').text()
            meetingGroupingSummaries = meetingGroupingSummaries || $('#meetingSummariesGroupContent').text()
            nextStepGroupingSummaries = nextStepGroupingSummaries || $('#nextStepGroupContent').text()

            // if all grouping summaries is empty, return empty string
            if (!overviewGroupingSummaries && !meetingGroupingSummaries && !nextStepGroupingSummaries) {
                isNotNull = false
            } else isNotNull = true
        } 
        
        if (!isNotNull) return ''

        const fullText = `# Group Meeting Summary\n\n` +
            `## Overview\n${overviewGroupingSummaries}\n\n` +
            `## Meeting Summaries\n${meetingGroupingSummaries}\n\n` +
            `## Next Steps\n${nextStepGroupingSummaries}\n\n` +
            `----------------------------------------\n` +
            `Generated by Go-Echo-Notes | ${new Date().toLocaleDateString()}`

        return fullText
    }

    function getFullTextSummariesData() {
        const name = $('#summariesNameDetailModal').text()
        const description = $('#summariesDescriptionDetailModal').text()
        const tldr = $('#tldrContent').text()
        const comprehensive = $('#comprehensiveContent').text()

        const fullText = `# Meeting Summary\n\n` +
            `## Name: ${name}\n\n` +
            `## Description: ${description}\n\n` +
            `## TLDR: ${tldr}\n\n` +
            `## Comprehensive Summary: ${comprehensive}\n\n` +
            `----------------------------------------\n` +
            `Generated by Go-Echo-Notes | ${new Date().toLocaleDateString()}`

        return fullText
    }

    // function to copy grouping summary to clipboard
    $('.copySummaryBtn').click(function () {
        // the flow is
        // trying to get full text grouping summaries first
        // if not found, get full text summaries data (so it will be used for single summaries)
        // if not found, show error message
        // if found, copy to clipboard
        let fullText = getFullTextGroupingSummaries()

        if (!fullText) fullText = getFullTextSummariesData()

        if (!fullText) {
            showInfoModal('Failed to copy text!', 'Copy Summary')
            return
        }

        navigator.clipboard.writeText(fullText).then(function () {
            showInfoModal('Grouping Summary copied to clipboard!', 'Copy Summary')
        }, function () {
            showInfoModal('Failed to copy text!', 'Copy Summary')
        })
    })

    // function to download grouping summary as text file
    $('.downloadSummaryBtn').click(function () {
        // the flow is
        // trying to get full text grouping summaries first
        // if not found, get full text summaries data (so it will be used for single summaries)
        // if not found, show error message
        // if found, copy to clipboard
        let fullText = getFullTextGroupingSummaries()

        if (!fullText) fullText = getFullTextSummariesData()

        if (!fullText) {
            showInfoModal('Failed to download text!', 'Download Summary')
            return
        }

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

    // ============= SUMMARIES FILTER

    // filter show room olderst/ newest
    $('#summariesFilter').on('change', async function () {
        SEARCH_SUMMARIES = $('#summariesSearch').val().trim()
        FILTER_SUMMARIES = $('#summariesFilter').val()

        loadSummaries()
    })

    // filter show room per page
    $('#summariesShowFilter').on('change', async function () {
        SEARCH_SUMMARIES = $('#summariesSearch').val().trim()
        FILTER_SUMMARIES = $('#summariesFilter').val()
        PER_PAGE = parseInt($('#summariesShowFilter').val())
        PAGE = 1

        loadSummaries()
    })

    // search room
    $('#summariesSearch').on('input', async function () {
        SEARCH_SUMMARIES = $('#summariesSearch').val().trim()
        FILTER_SUMMARIES = $('#summariesFilter').val()

        clearTimeout(SEARCH_TIMEOUT)

        SEARCH_TIMEOUT = setTimeout(async function () {
            PAGE = 1 // automatic set page to first page if using search filter
            loadSummaries()
        }, 1000)

    })

    // ============= LOGOUT BUTTON
    $('#logoutBtn').click(async function () {
        event.preventDefault()

        showLoader()

        try {
            const resp = await fetch("/api/logout", {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
            })
            const response = await resp.json()

            if (!response.error) {
                hideLoader()
                // setup for sso, just throw to dashboard page for now, because on backend will redirect to SSO dashboard page
                window.location.href = '/'

            } else {
                hideLoader()
                showInfoModal('Logout Failed', 'Failed to logout')
            }

        } catch (e) {
            hideLoader()
            showInfoModal('Logout Failed', 'Failed to logout')
        }

    })
})