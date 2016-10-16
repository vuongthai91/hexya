// Copyright 2016 NDP Systèmes. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"reflect"

	"github.com/npiganeau/yep/yep/tools/logging"
)

// Call calls the given method name methName on the given RecordCollection
// with the given arguments and return the result as interface{}.
func (rc RecordCollection) Call(methName string, args ...interface{}) interface{} {
	methInfo, ok := rc.mi.methods.get(methName)
	if !ok {
		logging.LogAndPanic(log, "Unknown method in model", "method", methName, "model", rc.mi.name)
	}
	methLayer := methInfo.topLayer
	return rc.call(methLayer, args...)
}

// Super calls the next method Layer.
// This method is meant to be used inside a method layer function to call its parent.
func (rc RecordCollection) Super(args ...interface{}) interface{} {
	if len(rc.callStack) == 0 {
		logging.LogAndPanic(log, "Empty call stack", "model", rc.mi.name)
	}
	currentLayer := rc.callStack[0]
	methInfo := currentLayer.methInfo
	methLayer := methInfo.getNextLayer(currentLayer)
	if methLayer == nil {
		// No parent
		return nil
	}

	return rc.call(methLayer, args...)
}

// MethodType returns the type of the method given by methName
func (rc RecordCollection) MethodType(methName string) reflect.Type {
	methInfo, ok := rc.mi.methods.get(methName)
	if !ok {
		logging.LogAndPanic(log, "Unknown method in model", "model", rc.mi.name, "method", methName)
	}
	return methInfo.methodType
}

// call is a wrapper around reflect.Value.Call() to use with interface{} type.
func (rc RecordCollection) call(methLayer *methodLayer, args ...interface{}) interface{} {
	rc.callStack = append([]*methodLayer{methLayer}, rc.callStack...)

	inVals := make([]reflect.Value, len(args)+1)
	inVals[0] = reflect.ValueOf(rc)
	for i, arg := range args {
		inVals[i+1] = reflect.ValueOf(arg)
	}

	var retVal []reflect.Value
	retVal = methLayer.funcValue.Call(inVals)

	if len(retVal) == 0 {
		return nil
	}
	return retVal[0].Interface()
}