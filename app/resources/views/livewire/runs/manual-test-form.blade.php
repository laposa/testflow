@php
    /** @var \App\Models\SessionServiceSuiteTest $test */
@endphp
<div class="manual-test">
    <div class="heading">
        <h2>{{ $test->getInstructions()->description }}</h2>
        <div class="description">
            {{ $test->getInstructions()->suite }}
        </div>
    </div>

    <table>
        <thead>
            <tr>
                <th>Step</th>
                <th>Action</th>
                <th>Expected result</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            @foreach($test->getInstructions()->steps as $index => $step)
                <tr>
                    <td>{{ $index + 1 }}</td>
                    <td>
                        @if(count($step->action) > 1)
                            <ul class="bullet-list">
                                @foreach($step->action as $action)
                                    <li>{{ $action }}</li>
                                @endforeach
                            </ul>
                        @else
                            {{ $step->action[0] ?? '' }}
                        @endif

                        @if($step->input)
                            <div class="input">
                                <span class="expand-content"></span>
                                <div class="expandable-content">
                                    @foreach($step->input as $key => $input)
                                        @if(is_array($input))
                                            <div><b>{{ $input['description'] }}: </b> <span class="copy-to-clipboard" title="Copy to clipboard">{{ isset($input['value']) ? $input['value'] : '' }}</span></div>
                                        @else
                                            <div><b>{{ $key }}</b>: {{ $input }}</div>
                                        @endif
                                    @endforeach
                                </div>
                            </div>
                        @endif
                    </td>
                    <td>
                        <ul class="bullet-list">
                            @foreach($step->result as $result)
                                <li>{{ $result }}</li>
                            @endforeach
                        </ul>
                    </td>
                    <td>
                        <div class="radio-wrapper">
                            <label for="test-step-{{$index}}-pass" class="custom-radio" title="Mark as Passed">
                                <input type="radio" name="test-step-{{$index}}" id="test-step-{{$index}}-pass" class="radio-pass" />
                                <div class="radio pass"></div>
                                Pass
                            </label>
        
                            <label for="test-step-{{$index}}-fail" class="custom-radio" title="Mark as Failed">
                                <input type="radio" name="test-step-{{$index}}" id="test-step-{{$index}}-fail" class="radio-fail"  />
                                <div class="radio fail"></div>
                                Fail
                            </label>
                        </div>
                    </td>
                </tr>
            @endforeach
        </tbody>
    </table>

    <form wire:submit="save">

        <div>
            <label for="comment">Comment</label>
            <textarea
                id="comment"
                name="comment"
                wire:model="comment"></textarea>
        </div>

        <div class="test-status">
            
            <label for="pass">
                <input 
                    type="radio"
                    id="pass"
                    name="result"
                    value="pass"
                    wire:model="result"
                    required
                />
                <div class="pass">
                    Pass
                </div>
            </label>
            
            <label for="fail">
                <input
                    type="radio"
                    id="fail"
                    value="fail"
                    name="result"
                    wire:model="result"
                    required
                />
                <div class="fail">
                    Fail
                </div>
            </label>
            
            <button type="submit" disabled>Submit</button>
        </div>
    </form>
</div>
