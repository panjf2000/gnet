/*
 * Copyright (c) 2023 The Gnet Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package gnet

import "github.com/panjf2000/gnet/v2/internal/netpoll"

func (eng *engine) accept(fd int, ev netpoll.IOEvent) error {
	return eng.accept1(fd, ev, 0)
}

func (el *eventloop) accept(fd int, ev netpoll.IOEvent) error {
	return el.accept1(fd, ev, 0)
}
